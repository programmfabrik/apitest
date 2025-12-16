package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/programmfabrik/apitest/internal/httpproxy"
	"github.com/programmfabrik/apitest/internal/smtp"
	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/programmfabrik/apitest/pkg/lib/report"
	"github.com/programmfabrik/apitest/pkg/lib/template"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

// Suite defines the structure of our apitest. We do read this in with the config loader
type Suite struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	HttpServer  *struct {
		Addr     string                `json:"addr"`
		Dir      string                `json:"dir"`
		Testmode bool                  `json:"testmode"`
		Proxy    httpproxy.ProxyConfig `json:"proxy"`
	} `json:"http_server,omitempty"`
	SmtpServer *struct {
		Addr           string `json:"addr"`
		MaxMessageSize int64  `json:"max_message_size"`
	} `json:"smtp_server,omitempty"`
	Tests []any          `json:"tests"`
	Store map[string]any `json:"store"`

	StandardHeader          map[string]any    `yaml:"header" json:"header"`
	StandardHeaderFromStore map[string]string `yaml:"header_from_store" json:"header_from_store"`

	config          testToolConfig
	datastore       *datastore.Datastore
	manifestRelDir  string
	manifestDir     string
	manifestPath    string
	reporterRoot    *report.ReportElement
	index           int
	serverURL       *url.URL
	httpServer      *http.Server
	httpServerProxy *httpproxy.Proxy
	httpServerDir   string
	idleConnsClosed chan struct{}
	httpServerHost  string
	loader          template.Loader
	smtpServer      *smtp.Server
}

// newTestSuite creates a new suite on which we execute our tests on. Normally this only gets call from within the apitest main command
func newTestSuite(
	config testToolConfig,
	manifestPath, manifestDir string,
	r *report.ReportElement,
	datastore *datastore.Datastore,
	index int,
) (suite *Suite, err error) {

	var (
		suitePreload Suite
	)

	suite = &Suite{
		config:         config,
		manifestDir:    filepath.Dir(manifestPath),
		manifestPath:   manifestPath,
		manifestRelDir: manifestDir,
		reporterRoot:   r,
		datastore:      datastore,
		index:          index,
	}

	// Here we create this additional struct in order to preload the suite manifest
	// It is needed, for example, for getting the suite HTTP server address
	// Then preloaded values are used to load again the manifest with relevant replacements
	suitePreload = Suite{
		config:         config,
		manifestDir:    filepath.Dir(manifestPath),
		manifestPath:   manifestPath,
		manifestRelDir: manifestDir,
		reporterRoot:   r,
		datastore:      datastore,
		index:          index,
	}

	manifest, err := suitePreload.loadManifest()
	if err != nil {
		err = fmt.Errorf("loading manifest: %w", err)
		suitePreload.reporterRoot.Failure = err.Error()
		return &suitePreload, err
	}

	err = jsutil.Unmarshal(manifest, &suitePreload)
	if err != nil {
		err = fmt.Errorf("unmarshaling manifest %q: %w", manifestPath, err)
		suitePreload.reporterRoot.Failure = err.Error()
		return &suitePreload, err
	}

	// Add external http server url here, as only after this point the http_server.addr may be available
	if httpServerReplaceHost != "" {
		_, err = url.Parse("//" + httpServerReplaceHost)
		if err != nil {
			return nil, fmt.Errorf("set http_server_host failed (command argument): %w", err)
		}
	}
	if suitePreload.HttpServer != nil {
		preloadHTTPAddrStr := suitePreload.HttpServer.Addr
		if preloadHTTPAddrStr == "" {
			preloadHTTPAddrStr = ":80"
		}
		// We need to append it as the golang URL parser is not smart enough to differenciate between hostname and protocol
		_, err = url.Parse("//" + preloadHTTPAddrStr)
		if err != nil {
			return nil, fmt.Errorf("set http_server_host failed (manifesr addr): %w", err)
		}
	}
	suitePreload.httpServerHost = httpServerReplaceHost

	// Here we load the usable manifest, now that we can do all potential replacements
	manifest, err = suitePreload.loadManifest()
	if err != nil {
		err = fmt.Errorf("loading manifest: %w", err)
		suite.reporterRoot.Failure = err.Error()
		return suite, err
	}
	// fmt.Printf(%q, string(manifest))
	// We unmarshall the final manifest into the final working suite
	err = jsutil.Unmarshal(manifest, &suite)
	if err != nil {
		err = fmt.Errorf("unmarshaling manifest %q: %w", manifestPath, err)
		suite.reporterRoot.Failure = err.Error()
		return suite, err
	}
	suite.httpServerHost = suitePreload.httpServerHost
	suite.loader = suitePreload.loader

	// Append suite manifest path to name, so we know in an automatic setup where the test is loaded from
	suite.Name = fmt.Sprintf("%s (%s)", suite.Name, manifestPath)

	// Parse serverURL
	suite.serverURL, err = url.Parse(suite.config.serverURL)
	if err != nil {
		return nil, fmt.Errorf("can not load server url : %w", err)
	}

	// init store
	err = suite.datastore.SetMap(suite.Store)
	if err != nil {
		err = fmt.Errorf("setting datastore map: %w", err)
		suite.reporterRoot.Failure = err.Error()
		return suite, err
	}

	return suite, nil
}

// run run the given testsuite
func (ats *Suite) run() bool {
	r := ats.reporterRoot
	if !ats.config.logShort {
		logrus.Infof("[%2d] '%s'", ats.index, ats.Name)
	}

	ats.startSmtpServer()
	defer ats.stopSmtpServer()

	ats.startHttpServer()
	defer ats.stopHttpServer()

	err := os.Chdir(ats.manifestDir)
	if err != nil {
		logrus.Fatalf("Unable to switch working directory to %q", ats.manifestDir)
	}

	start := time.Now()

	success := true
	for k, v := range ats.Tests {
		child := r.NewChild(strconv.Itoa(k))

		sTestSuccess := ats.parseAndRunTest(
			v,
			ats.manifestPath,
			child,
			ats.loader,
			true, // parallel exec allowed for top-level tests
		)

		child.Leave(sTestSuccess)

		if !sTestSuccess {
			success = false
			break
		}
	}

	elapsed := time.Since(start)
	r.Leave(success)
	if success {
		if ats.config.logShort {
			fmt.Printf("OK '%s' (%.3fs)\n", ats.manifestRelDir, elapsed.Seconds())
		} else {
			logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Infof("[%2d] success", ats.index)
		}
	} else {
		if ats.config.logShort {
			fmt.Printf("FAIL '%s' (%.3fs)\n", ats.manifestRelDir, elapsed.Seconds())
		} else {
			logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Warnf("[%2d] failure", ats.index)
		}
	}

	if keepRunning { // flag defined in main.go
		logrus.Info("Waiting until a keyboard interrupt (usually CTRL+C) is received...")

		if ats.HttpServer != nil {
			logrus.Info("HTTP Server URL:", ats.HttpServer.Addr)
		}
		if ats.SmtpServer != nil {
			logrus.Info("SMTP Server URL:", ats.SmtpServer.Addr)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		<-sigChan
	}

	return success
}

type testContainer struct {
	CaseByte []byte
	Path     string
}

func (ats *Suite) buildLoader(rootLoader template.Loader, parallelRunIdx int) template.Loader {
	loader := template.NewLoader(ats.datastore)
	loader.Delimiters = rootLoader.Delimiters
	loader.HTTPServerHost = ats.httpServerHost
	loader.ServerURL = ats.serverURL
	loader.OAuthClient = ats.config.oAuthClient

	if rootLoader.ParallelRunIdx < 0 {
		loader.ParallelRunIdx = parallelRunIdx
	} else {
		loader.ParallelRunIdx = rootLoader.ParallelRunIdx
	}

	return loader
}

func (ats *Suite) parseAndRunTest(
	v any,
	testFilePath string,
	r *report.ReportElement,
	rootLoader template.Loader,
	allowParallelExec bool,
) bool {
	parallelRuns := 1

	// Get the Manifest with @ logic
	referencedPathSpec, testRaw, err := template.LoadManifestDataAsRawJson(v, filepath.Dir(testFilePath))
	if err != nil {
		r.SaveToReportLog(err.Error())
		logrus.Error(fmt.Errorf("can not LoadManifestDataAsRawJson (%s): %w", testFilePath, err))
		return false
	}
	if referencedPathSpec != nil {
		testFilePath = filepath.Join(filepath.Dir(testFilePath), referencedPathSpec.Path)
		parallelRuns = referencedPathSpec.ParallelRuns
	}

	// If parallel runs are requested, check that they're actually allowed
	if parallelRuns > 1 && !allowParallelExec {
		logrus.Error(fmt.Errorf("parallel runs are not allowed in nested tests in (%s)", testFilePath))
		return false
	}

	// Execute test cases
	var successCount atomic.Uint32
	var waitGroup sync.WaitGroup

	waitGroup.Add(parallelRuns)

	for runIdx := range parallelRuns {
		go ats.testGoroutine(
			&waitGroup, &successCount, testFilePath, r, rootLoader,
			runIdx, testRaw,
		)
	}

	waitGroup.Wait()

	return successCount.Load() == uint32(parallelRuns)
}

func (ats *Suite) testGoroutine(
	waitGroup *sync.WaitGroup,
	successCount *atomic.Uint32,
	testFilePath string,
	r *report.ReportElement,
	rootLoader template.Loader,
	runIdx int,
	testRaw jsutil.RawMessage,
) {
	defer waitGroup.Done()

	testFileDir := filepath.Dir(testFilePath)

	// Build template loader
	loader := ats.buildLoader(rootLoader, runIdx)

	// Parse testRaw as template
	testRendered, err := loader.Render(testRaw, testFileDir, nil)
	if err != nil {
		r.SaveToReportLog(err.Error())
		logrus.Error(fmt.Errorf("can not render template (%s): %w", testFilePath, err))

		// note that successCount is not incremented
		return
	}

	// Build list of test cases
	var testCases []jsutil.RawMessage
	err = jsutil.Unmarshal(testRendered, &testCases)
	if err != nil {
		// Input could not be deserialized into list, try to deserialize into single object
		var singleTest jsutil.RawMessage
		err = jsutil.Unmarshal(testRendered, &singleTest)
		if err != nil {
			// Malformed json
			r.SaveToReportLog(err.Error())
			logrus.Error(fmt.Errorf("can not unmarshal (%s): %w", testFilePath, err))

			// note that successCount is not incremented
			return
		}

		testCases = []jsutil.RawMessage{singleTest}
	}

	for testIdx, testCase := range testCases {
		var success bool

		// If testCase can be unmarshalled as string, we may have a
		// reference to another test using @ notation at hand
		var testCaseStr string
		err = jsutil.Unmarshal(testCase, &testCaseStr)
		if err == nil && util.IsPathSpec(testCaseStr) {
			// Recurse if the testCase points to another file using @ notation
			success = ats.parseAndRunTest(
				testCaseStr,
				testFilePath,
				r,
				loader,
				false, // no parallel exec allowed in nested tests
			)
		} else {
			// Otherwise simply run the literal test case
			success = ats.runLiteralTest(
				testContainer{
					CaseByte: testCase,
					Path:     testFileDir,
				},
				r,
				testFilePath,
				loader,
				runIdx*len(testCases)+testIdx,
			)
		}

		if !success {
			// note that successCount is not incremented
			return
		}
	}

	successCount.Add(1)
}

func (ats *Suite) runLiteralTest(
	tc testContainer,
	r *report.ReportElement,
	testFilePath string,
	loader template.Loader,
	index int,
) bool {
	r.SetName(testFilePath)

	var test Case
	err := jsutil.Unmarshal(tc.CaseByte, &test)
	if err != nil {
		r.SaveToReportLog(err.Error())
		logrus.Error(fmt.Errorf("can not unmarshal single test (%s): %w", testFilePath, err))
		return false
	}

	test.Filename = testFilePath
	test.loader = loader
	test.manifestDir = tc.Path
	test.suiteIndex = ats.index
	test.index = index
	test.dataStore = ats.datastore
	test.standardHeader = ats.StandardHeader
	test.standardHeaderFromStore = ats.StandardHeaderFromStore
	if test.LogNetwork == nil {
		test.LogNetwork = &ats.config.logNetwork
	}
	if test.LogVerbose == nil {
		test.LogVerbose = &ats.config.logVerbose
	}
	if test.LogShort == nil {
		test.LogShort = &ats.config.logShort
	}
	if test.ServerURL == "" {
		test.ServerURL = ats.config.serverURL
	}
	success := test.runAPITestCase(r)

	if !success && !test.ContinueOnFailure {
		return false
	}

	return true
}

func (ats *Suite) loadManifest() (manifest []byte, err error) {
	var (
		loader       template.Loader
		serverURL    *url.URL
		manifestFile afero.File
	)

	if !ats.config.logShort {
		logrus.Tracef("Loading manifest: %s", ats.manifestPath)
	}

	loader = template.NewLoader(ats.datastore)
	loader.HTTPServerHost = ats.httpServerHost
	serverURL, err = url.Parse(ats.config.serverURL)
	if err != nil {
		return nil, fmt.Errorf("can not load server url into manifest (%s): %w", ats.manifestPath, err)
	}
	loader.ServerURL = serverURL
	loader.OAuthClient = ats.config.oAuthClient

	manifestFile, err = filesystem.Fs.Open(ats.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("opening manifestPath (%s): %w", ats.manifestPath, err)
	}
	defer manifestFile.Close()

	manifestTmpl, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("loading manifest (%s): %w", ats.manifestPath, err)
	}

	manifest, err = loader.Render(manifestTmpl, ats.manifestDir, nil)
	if err != nil {
		return nil, fmt.Errorf("loading manifest (%s): %w", ats.manifestPath, err)
	}

	ats.loader = loader
	return manifest, nil
}
