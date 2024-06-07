package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/programmfabrik/apitest/internal/httpproxy"
	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
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
	Tests []any          `json:"tests"`
	Store map[string]any `json:"store"`

	StandardHeader          map[string]*string `yaml:"header" json:"header"`
	StandardHeaderFromStore map[string]string  `yaml:"header_from_store" json:"header_from_store"`

	Config          TestToolConfig
	datastore       *datastore.Datastore
	manifestRelDir  string
	manifestDir     string
	manifestPath    string
	reporterRoot    *report.ReportElement
	index           int
	serverURL       string
	httpServer      http.Server
	httpServerProxy *httpproxy.Proxy
	httpServerDir   string
	idleConnsClosed chan struct{}
	HTTPServerHost  string
	loader          template.Loader
}

// NewTestSuite creates a new suite on which we execute our tests on. Normally this only gets call from within the apitest main command
func NewTestSuite(config TestToolConfig, manifestPath string, manifestDir string, r *report.ReportElement, datastore *datastore.Datastore, index int) (*Suite, error) {
	suite := Suite{
		Config:         config,
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
	suitePreload := Suite{
		Config:         config,
		manifestDir:    filepath.Dir(manifestPath),
		manifestPath:   manifestPath,
		manifestRelDir: manifestDir,
		reporterRoot:   r,
		datastore:      datastore,
		index:          index,
	}
	manifest, err := suitePreload.loadManifest()
	if err != nil {
		err = fmt.Errorf("error loading manifest: %s", err)
		suitePreload.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suitePreload, err
	}

	err = util.Unmarshal(manifest, &suitePreload)
	if err != nil {
		err = fmt.Errorf("error unmarshaling manifest '%s': %s", manifestPath, err)
		suitePreload.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suitePreload, err
	}

	// Add external http server url here, as only after this point the http_server.addr may be available
	hsru := new(url.URL)
	shsu := new(url.URL)
	if httpServerReplaceHost != "" {
		hsru, err = url.Parse("//" + httpServerReplaceHost)
		if err != nil {
			return nil, errors.Wrap(err, "set http_server_host failed (command argument)")
		}
	}
	if suitePreload.HttpServer != nil {
		preloadHTTPAddrStr := suitePreload.HttpServer.Addr
		if preloadHTTPAddrStr == "" {
			preloadHTTPAddrStr = ":80"
		}
		// We need to append it as the golang URL parser is not smart enough to differenciate between hostname and protocol
		shsu, err = url.Parse("//" + preloadHTTPAddrStr)
		if err != nil {
			return nil, errors.Wrap(err, "set http_server_host failed (manifesr addr)")
		}
	}
	suitePreload.HTTPServerHost = ""
	if hsru.Hostname() != "" {
		suitePreload.HTTPServerHost = hsru.Hostname()
	} else if shsu.Hostname() != "" {
		suitePreload.HTTPServerHost = shsu.Hostname()
	} else {
		suitePreload.HTTPServerHost = "localhost"
	}
	if suite.HTTPServerHost == "0.0.0.0" {
		suitePreload.HTTPServerHost = "localhost"
	}
	if hsru.Port() != "" {
		suitePreload.HTTPServerHost += ":" + hsru.Port()
	} else if shsu.Port() != "" {
		suitePreload.HTTPServerHost += ":" + shsu.Port()
	}

	// Here we load the usable manifest, now that we can do all potential replacements
	manifest, err = suitePreload.loadManifest()
	if err != nil {
		err = fmt.Errorf("error loading manifest: %s", err)
		suite.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suite, err
	}
	// fmt.Printf("%s", string(manifest))
	// We unmarshall the final manifest into the final working suite
	err = util.Unmarshal(manifest, &suite)
	if err != nil {
		err = fmt.Errorf("error unmarshaling manifest '%s': %s", manifestPath, err)
		suite.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suite, err
	}
	suite.HTTPServerHost = suitePreload.HTTPServerHost
	suite.loader = suitePreload.loader

	//Append suite manifest path to name, so we know in an automatic setup where the test is loaded from
	suite.Name = fmt.Sprintf("%s (%s)", suite.Name, manifestPath)

	// init store
	err = suite.datastore.SetMap(suite.Store)
	if err != nil {
		err = fmt.Errorf("error setting datastore map:%s", err)
		suite.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suite, err
	}

	return &suite, nil
}

// Run run the given testsuite
func (ats *Suite) Run() bool {
	r := ats.reporterRoot
	if !ats.Config.LogShort {
		logrus.Infof("[%2d] '%s'", ats.index, ats.Name)
	}

	ats.StartHttpServer()

	err := os.Chdir(ats.manifestDir)
	if err != nil {
		logrus.Errorf("Unable to switch working directory to %q", ats.manifestDir)
	}

	start := time.Now()

	success := true
	for k, v := range ats.Tests {
		child := r.NewChild(strconv.Itoa(k))

		sTestSuccess := ats.parseAndRunTest(
			v,
			ats.manifestDir,
			ats.manifestPath,
			child,
			ats.loader,
			true,  // parallel exec allowed for top-level tests
			false, // don't force ContinueOnFail at this point
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
		if ats.Config.LogShort {
			fmt.Printf("OK '%s' (%.3fs)\n", ats.manifestRelDir, elapsed.Seconds())
		} else {
			logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Infof("[%2d] success", ats.index)
		}
	} else {
		if ats.Config.LogShort {
			fmt.Printf("FAIL '%s' (%.3fs)\n", ats.manifestRelDir, elapsed.Seconds())
		} else {
			logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Warnf("[%2d] failure", ats.index)
		}
	}

	ats.StopHttpServer()

	return success
}

type TestContainer struct {
	CaseByte json.RawMessage
	Path     string
}

func (ats *Suite) parseAndRunTest(
	v any, manifestDir, testFilePath string, r *report.ReportElement,
	rootLoader template.Loader, allowParallelExec bool, forceContinueOnFail bool,
) bool {
	//Init variables
	// logrus.Warnf("Test %s, Prev delimiters: %#v", testFilePath, rootLoader.Delimiters)
	loader := template.NewLoader(ats.datastore)
	loader.Delimiters = rootLoader.Delimiters
	loader.HTTPServerHost = ats.HTTPServerHost
	serverURL, err := url.Parse(ats.Config.ServerURL)
	if err != nil {
		logrus.Error(fmt.Errorf("can not load server url into test (%s): %s", testFilePath, err))
		return false
	}
	loader.ServerURL = serverURL
	loader.OAuthClient = ats.Config.OAuthClient

	// Determine number of parallel repetitions, if any
	repetitions := 1
	if vStr, ok := v.(string); ok {
		pathSpec, ok := util.ParsePathSpec(vStr)
		if ok {
			repetitions = pathSpec.Repetitions
		}
	}

	// Configure parallel repetitions, if specified
	if repetitions > 1 {
		// Check that parallel repetitions are actually allowed
		if !allowParallelExec {
			logrus.Error(fmt.Errorf("parallel repetitions are not allowed in nested tests (%s)", testFilePath))
			return false
		}

		// If we're running in parallel repetition mode, force subordinate tests to ContinueOnFail
		forceContinueOnFail = true
	}

	// Get the Manifest with @ logic
	fileh, testObj, err := template.LoadManifestDataAsRawJson(v, manifestDir)
	dir := filepath.Dir(fileh)
	if fileh != "" {
		testFilePath = filepath.Join(filepath.Dir(testFilePath), fileh)
	}
	if err != nil {
		r.SaveToReportLog(err.Error())
		logrus.Error(fmt.Errorf("can not LoadManifestDataAsRawJson (%s): %s", testFilePath, err))
		return false
	}

	// Parse as template always
	testObj, err = loader.Render(testObj, filepath.Join(manifestDir, dir), nil)
	if err != nil {
		r.SaveToReportLog(err.Error())
		logrus.Error(fmt.Errorf("can not render template (%s): %s", testFilePath, err))
		return false
	}

	// Build list of test cases
	var testCases []json.RawMessage
	err = util.Unmarshal(testObj, &testCases)
	if err != nil {
		// Input could not be deserialized into list, try to deserialize into single object
		var singleTest json.RawMessage
		err = util.Unmarshal(testObj, &singleTest)
		if err != nil {
			// Malformed json
			r.SaveToReportLog(err.Error())
			logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
			return false
		}

		testCases = []json.RawMessage{singleTest}
	}

	// Execute test cases
	successCh := make(chan bool, repetitions)

	for repeatIdx := range repetitions {
		go func() {
			for testIdx, testCase := range testCases {
				var success bool

				// If testCase can be unmarshalled as string, we may have a
				// reference to another test using @ notation at hand
				var testCaseStr string
				err = util.Unmarshal(testCase, &testCaseStr)
				if err == nil && util.IsPathSpec(testCaseStr) {
					// Recurse if the testCase points to another file using @ notation
					success = ats.parseAndRunTest(
						testCaseStr,
						filepath.Join(manifestDir, dir),
						testFilePath,
						r,
						loader,
						false, // no parallel exec allowed in nested tests
						forceContinueOnFail,
					)
				} else {
					// Otherwise simply run the literal test case
					success = ats.runLiteralTest(
						TestContainer{
							CaseByte: testCase,
							Path:     filepath.Join(manifestDir, dir),
						},
						r,
						testFilePath,
						loader,
						repeatIdx*len(testCases)+testIdx,
						forceContinueOnFail,
					)
				}

				if !success {
					successCh <- false
					return
				}
			}

			successCh <- true
		}()
	}

	for range repetitions {
		if success := <-successCh; !success {
			return false
		}
	}

	return true
}

func (ats *Suite) runLiteralTest(
	tc TestContainer, r *report.ReportElement, testFilePath string, loader template.Loader,
	index int, forceContinueOnFailure bool,
) bool {
	r.SetName(testFilePath)

	var test Case
	jErr := util.Unmarshal(tc.CaseByte, &test)
	if jErr != nil {

		r.SaveToReportLog(jErr.Error())
		logrus.Error(fmt.Errorf("can not unmarshal single test (%s): %s", testFilePath, jErr))

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
	if forceContinueOnFailure {
		test.ContinueOnFailure = true
	}
	if test.LogNetwork == nil {
		test.LogNetwork = &ats.Config.LogNetwork
	}
	if test.LogVerbose == nil {
		test.LogVerbose = &ats.Config.LogVerbose
	}
	if test.LogShort == nil {
		test.LogShort = &ats.Config.LogShort
	}
	if test.ServerURL == "" {
		test.ServerURL = ats.Config.ServerURL
	}
	success := test.runAPITestCase(r)

	if !success && !test.ContinueOnFailure {
		return false
	}

	return true
}

func (ats *Suite) loadManifest() ([]byte, error) {
	var res []byte
	if !ats.Config.LogShort {
		logrus.Tracef("Loading manifest: %s", ats.manifestPath)
	}
	loader := template.NewLoader(ats.datastore)
	loader.HTTPServerHost = ats.HTTPServerHost
	serverURL, err := url.Parse(ats.Config.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("can not load server url into manifest (%s): %s", ats.manifestPath, err)
	}
	loader.ServerURL = serverURL
	loader.OAuthClient = ats.Config.OAuthClient
	manifestFile, err := filesystem.Fs.Open(ats.manifestPath)
	if err != nil {
		return res, fmt.Errorf("error opening manifestPath (%s): %s", ats.manifestPath, err)
	}
	defer manifestFile.Close()

	manifestTmpl, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return res, fmt.Errorf("error loading manifest (%s): %s", ats.manifestPath, err)
	}

	b, err := loader.Render(manifestTmpl, ats.manifestDir, nil)
	ats.loader = loader
	return b, err
}
