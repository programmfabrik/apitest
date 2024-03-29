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
		sTestSuccess := ats.parseAndRunTest(v, ats.manifestDir, ats.manifestPath, k, 1, false, child, ats.loader)
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

func (ats *Suite) parseAndRunTest(v any, manifestDir, testFilePath string, k, repeatNTimes int, runParallel bool, r *report.ReportElement, rootLoader template.Loader) bool {
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

	isParallelPathSpec := false
	switch t := v.(type) {
	case string:
		isParallelPathSpec = util.IsParallelPathSpec(t)
	}

	//Get the Manifest with @ logic
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

	//Try to directly unmarshal the manifest into testcase array
	var testCases []json.RawMessage
	err = util.Unmarshal(testObj, &testCases)
	if err == nil {
		d := 1
		if isParallelPathSpec || runParallel {
			if repeatNTimes > 1 {
				logrus.Debugf("run %s parallel: repeat %d times", filepath.Base(testFilePath), repeatNTimes)
			}
			d = len(testCases)
		}

		waitCh := make(chan bool, repeatNTimes*d)
		succCh := make(chan bool, repeatNTimes*len(testCases))

		go func() {
			for kn := 0; kn < repeatNTimes; kn++ {
				for ki, v := range testCases {
					waitCh <- true
					go testGoRoutine(k, kn+ki*repeatNTimes, v, ats, testFilePath, manifestDir, dir, r, loader, waitCh, succCh, isParallelPathSpec || runParallel)
				}
			}
		}()

		for i := 0; i < repeatNTimes*len(testCases); i++ {
			select {
			case succ := <-succCh:
				if succ == false {
					return false
				}
			}
		}
	} else {
		// We were not able unmarshal into array, so we try to unmarshal into raw message

		// Get the (optional) number of repititions from the test path spec
		parallelRepititions := 1
		if isParallelPathSpec {
			switch t := v.(type) {
			case string:
				parallelRepititions, _ = util.GetParallelPathSpec(t)
			}
		}

		// Parse as template always
		requestBytes, lErr := loader.Render(testObj, filepath.Join(manifestDir, dir), nil)
		if lErr != nil {
			r.SaveToReportLog(lErr.Error())
			logrus.Error(fmt.Errorf("can not render template (%s): %s", testFilePath, lErr))
			return false
		}

		// If objects are different, we did have a Go template, recurse one level deep
		if string(requestBytes) != string(testObj) {
			return ats.parseAndRunTest([]byte(requestBytes), filepath.Join(manifestDir, dir),
				testFilePath, k, parallelRepititions, isParallelPathSpec, r, loader)
		}

		// Its a JSON at this point, assign and proceed to parse
		testObj = requestBytes

		var singleTest json.RawMessage
		err = util.Unmarshal(testObj, &singleTest)
		if err == nil {

			//Check if is @ and if so load the test
			if util.IsPathSpec(string(testObj)) {
				var sS string

				err := util.Unmarshal(testObj, &sS)
				if err != nil {
					r.SaveToReportLog(err.Error())
					logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
					return false
				}

				return ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), testFilePath, k, parallelRepititions, isParallelPathSpec, r, template.Loader{})
			} else {
				return ats.runSingleTest(TestContainer{CaseByte: testObj, Path: filepath.Join(manifestDir, dir)}, r, testFilePath, loader, k, runParallel)
			}
		} else {
			// Malformed json
			r.SaveToReportLog(err.Error())
			logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
			return false
		}
	}

	return true
}

func (ats *Suite) runSingleTest(tc TestContainer, r *report.ReportElement, testFilePath string, loader template.Loader, k int, isParallel bool) bool {
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
	test.index = k
	test.dataStore = ats.datastore
	test.standardHeader = ats.StandardHeader
	test.standardHeaderFromStore = ats.StandardHeaderFromStore
	if isParallel {
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

func testGoRoutine(k, ki int, v json.RawMessage, ats *Suite, testFilePath, manifestDir, dir string, r *report.ReportElement, loader template.Loader, waitCh, succCh chan bool, runParallel bool) {
	success := false

	//Check if is @ and if so load the test
	switch util.IsPathSpec(string(v)) {
	case true:
		var sS string
		err := util.Unmarshal(v, &sS)
		if err != nil {
			r.SaveToReportLog(err.Error())
			logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
			success = false
			break
		}
		success = ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), testFilePath, k+ki, 1, runParallel, r, loader)
	default:
		success = ats.runSingleTest(TestContainer{CaseByte: v, Path: filepath.Join(manifestDir, dir)},
			r, testFilePath, loader, ki, runParallel)
	}

	succCh <- success
	if success {
		<-waitCh
	}
}
