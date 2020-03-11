package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/programmfabrik/apitest/pkg/lib/cjson"
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
		Addr     string `json:"addr"`
		Dir      string `json:"dir"`
		Testmode bool   `json:"testmode"`
	} `json:"http_server,omitempty"`
	Tests []interface{}          `json:"tests"`
	Store map[string]interface{} `json:"store"`

	StandardHeader          map[string]*string `yaml:"header" json:"header"`
	StandardHeaderFromStore map[string]string  `yaml:"header_from_store" json:"header_from_store"`

	Config          TestToolConfig
	datastore       *datastore.Datastore
	manifestDir     string
	manifestPath    string
	reporterRoot    *report.ReportElement
	index           int
	serverURL       string
	httpServer      http.Server
	httpServerDir   string
	idleConnsClosed chan struct{}
}

// NewTestSuite creates a new suite on which we execute our tests on. Normally this only gets call from within the apitest main command
func NewTestSuite(config TestToolConfig, manifestPath string, r *report.ReportElement, datastore *datastore.Datastore, index int) (*Suite, error) {
	suite := Suite{
		Config:       config,
		manifestDir:  filepath.Dir(manifestPath),
		manifestPath: manifestPath,
		reporterRoot: r,
		datastore:    datastore,
		index:        index,
	}

	manifest, err := suite.loadManifest()
	if err != nil {
		err = fmt.Errorf("error loading manifest: %s", err)
		suite.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suite, err
	}

	err = cjson.Unmarshal(manifest, &suite)
	if err != nil {
		err = fmt.Errorf("error unmarshaling manifest '%s': %s", manifestPath, err)
		suite.reporterRoot.Failure = fmt.Sprintf("%s", err)
		return &suite, err
	}

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

// StartHttpServer start a simple http server that can server local test resources during the testsuite is running
func (ats *Suite) StartHttpServer() {

	if ats.HttpServer == nil {
		return
	}

	ats.idleConnsClosed = make(chan struct{})
	mux := http.NewServeMux()

	if ats.HttpServer.Dir == "" {
		ats.httpServerDir = ats.manifestDir
	} else {
		ats.httpServerDir = filepath.Clean(ats.manifestDir + "/" + ats.HttpServer.Dir)
	}
	mux.Handle("/", http.FileServer(http.Dir(ats.httpServerDir)))

	ats.httpServer = http.Server{
		Addr:    ats.HttpServer.Addr,
		Handler: mux,
	}

	run := func() {
		logrus.Infof("Starting HTTP Server: %s: %s", ats.HttpServer.Addr, ats.httpServerDir)

		err := ats.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			// Error starting or closing listener:
			logrus.Errorf("HTTP server ListenAndServe: %v", err)
			return
		}
	}

	if ats.HttpServer.Testmode {
		// Run in foreground to test
		logrus.Infof("Testmode for HTTP Server. Listening, not running tests...")
		run()
	} else {
		go run()
	}
}

// StopHttpServer stop the http server that was started for this test suite
func (ats *Suite) StopHttpServer() {

	if ats.HttpServer == nil {
		return
	}

	err := ats.httpServer.Shutdown(context.Background())
	if err != nil {
		// Error from closing listeners, or context timeout:
		logrus.Errorf("HTTP server Shutdown: %v", err)
		close(ats.idleConnsClosed)
		<-ats.idleConnsClosed
	} else {
		logrus.Infof("Http Server stopped: %s", ats.httpServerDir)
	}
	return
}

// Run run the given testsuite
func (ats *Suite) Run() bool {
	r := ats.reporterRoot
	logrus.Infof("[%2d] '%s'", ats.index, ats.Name)

	ats.StartHttpServer()

	start := time.Now()

	success := true
	for k, v := range ats.Tests {
		child := r.NewChild(strconv.Itoa(k))
		sTestSuccess := ats.parseAndRunTest(v, ats.manifestDir, ats.manifestPath, k, false, child)
		child.Leave(sTestSuccess)
		if !sTestSuccess {
			success = false
			break
		}
	}

	elapsed := time.Since(start)
	r.Leave(success)
	if success {
		logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Infof("[%2d] success", ats.index)
	} else {
		logrus.WithFields(logrus.Fields{"elapsed": elapsed.Seconds()}).Warnf("[%2d] failure", ats.index)
	}

	ats.StopHttpServer()

	return success
}

type TestContainer struct {
	CaseByte json.RawMessage
	Path     string
}

func (ats *Suite) parseAndRunTest(v interface{}, manifestDir, testFilePath string, k int, runParallel bool, r *report.ReportElement) bool {
	//Init variables
	loader := template.NewLoader(ats.datastore)

	isParallelPathSpec := false
	switch t := v.(type) {
	case string:
		isParallelPathSpec = util.IsParallelPathSpec([]byte(t))
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
	err = cjson.Unmarshal(testObj, &testCases)
	if err == nil {
		d := 1
		if isParallelPathSpec || runParallel {
			d = len(testCases)
		}

		waitCh := make(chan bool, d)
		succCh := make(chan bool, len(testCases))

		go func() {
			for ki, v := range testCases {
				waitCh <- true
				go testGoRoutine(k, ki, v, ats, testFilePath, manifestDir, dir, r, loader, waitCh, succCh, isParallelPathSpec || runParallel)
			}
		}()

		for i := 0; i < len(testCases); i++ {
			select {
			case succ := <-succCh:
				if succ == false {
					return false
				}
			}
		}
	} else {
		//We were not able unmarshal into array, so we try to unmarshal into raw message
		var singleTest json.RawMessage
		err = cjson.Unmarshal(testObj, &singleTest)
		if err == nil {
			//Did work to unmarshal -> no go template

			//Check if is @ and if so load the test
			if util.IsPathSpec(testObj) {
				var sS string

				err := cjson.Unmarshal(testObj, &sS)
				if err != nil {
					r.SaveToReportLog(err.Error())
					logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
					return false
				}

				return ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), testFilePath, k, isParallelPathSpec, r)
			} else {
				return ats.runSingleTest(TestContainer{CaseByte: testObj, Path: filepath.Join(manifestDir, dir)}, r, testFilePath, loader, k, runParallel)
			}
		} else {
			//Did not work -> Could be go template or a mallformed json
			requestBytes, lErr := loader.Render(testObj, filepath.Join(manifestDir, dir), nil)
			if lErr != nil {
				r.SaveToReportLog(lErr.Error())
				logrus.Error(fmt.Errorf("can not render template (%s): %s", testFilePath, lErr))
				return false
			}

			//If the both objects are the same we did not have a template, but a mallformed json -> Call error
			if string(requestBytes) == string(testObj) {
				r.SaveToReportLog(err.Error())
				logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
				return false
			}

			//We have a template -> One level deeper with rendered bytes
			return ats.parseAndRunTest([]byte(requestBytes), filepath.Join(manifestDir, dir),
				testFilePath, k, isParallelPathSpec, r)

		}
	}

	return true
}

func (ats *Suite) runSingleTest(tc TestContainer, r *report.ReportElement, testFilePath string, loader template.Loader, k int, isParallel bool) bool {
	r.SetName(testFilePath)

	var test Case
	jErr := cjson.Unmarshal(tc.CaseByte, &test)
	if jErr != nil {

		r.SaveToReportLog(jErr.Error())
		logrus.Error(fmt.Errorf("can not unmarshal single test (%s): %s", testFilePath, jErr))

		return false
	}

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
	logrus.Tracef("Loading manifest: %s", ats.manifestPath)
	loader := template.NewLoader(ats.datastore)
	manifestFile, err := filesystem.Fs.Open(ats.manifestPath)
	if err != nil {
		return res, fmt.Errorf("error opening manifestPath (%s): %s", ats.manifestPath, err)
	}
	defer manifestFile.Close()

	manifestTmpl, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return res, fmt.Errorf("error loading manifest (%s): %s", ats.manifestPath, err)
	}
	return loader.Render(manifestTmpl, ats.manifestDir, nil)
}

func testGoRoutine(k, ki int, v json.RawMessage, ats *Suite, testFilePath, manifestDir, dir string, r *report.ReportElement, loader template.Loader, waitCh, succCh chan bool, runParallel bool) {
	success := false

	//Check if is @ and if so load the test
	switch util.IsPathSpec(v) {
	case true:
		var sS string
		err := cjson.Unmarshal(v, &sS)
		if err != nil {
			r.SaveToReportLog(err.Error())
			logrus.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
			success = false
			break
		}
		success = ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), testFilePath, k+ki, runParallel, r)
	default:
		success = ats.runSingleTest(TestContainer{CaseByte: v, Path: filepath.Join(manifestDir, dir)},
			r, testFilePath, loader, ki, runParallel)
	}

	succCh <- success
	if success {
		<-waitCh
	}
}
