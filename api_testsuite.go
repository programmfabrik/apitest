package main

import (
	"encoding/json"
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/datastore"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"

	"github.com/programmfabrik/fylr-apitest/lib/util"

	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"github.com/programmfabrik/fylr-apitest/lib/template"
	log "github.com/sirupsen/logrus"
)

// Suite defines the structure of our apitest
// We do read this in with the config loader
type Suite struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tests       []util.GenericJson     `json:"tests"`
	Store       map[string]interface{} `json:"store"`

	StandardHeader          map[string]*string `yaml:"header" json:"header"`
	StandardHeaderFromStore map[string]string  `yaml:"header_from_store" json:"header_from_store"`

	Config       TestToolConfig
	datastore    *datastore.Datastore
	manifestDir  string
	manifestPath string
	reporterRoot *report.ReportElement
	index        int
	serverURL    string
}

// NewTestSuite creates a new suite on which we execute our tests on
// Normally this only gets call from within the apitest main command
func NewTestSuite(
	config TestToolConfig,
	manifestPath string,
	r *report.ReportElement,
	datastore *datastore.Datastore,
	index int,
) (suite Suite, err error) {
	suite = Suite{
		Config:       config,
		manifestDir:  filepath.Dir(manifestPath),
		manifestPath: manifestPath,
		reporterRoot: r,
		datastore:    datastore,
		index:        index,
	}

	manifest, err := suite.loadManifest()
	if err != nil {
		return suite, fmt.Errorf("error loading manifest: %s", err)
	}

	if err = cjson.Unmarshal(manifest, &suite); err != nil {
		return suite, fmt.Errorf("error unmarshaling manifest '%s': %s", manifestPath, err)
	}

	//Append suite manifest path to name, so we know in an automatic setup where the test is loaded from
	suite.Name = fmt.Sprintf("%s (%s)", suite.Name, manifestPath)

	// init store
	err = suite.datastore.SetMap(suite.Store)
	if err != nil {
		err = fmt.Errorf("error setting datastore map:%s", err)
	}

	return suite, err
}

func (ats Suite) Run() (success bool) {
	r := ats.reporterRoot
	log.Infof("[%2d] '%s'", ats.index, ats.Name)

	//r.NewChild(ats.Name)
	//r.SetTestCount(len(ats.Tests))

	start := time.Now()

	success = true
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
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Infof("[%2d] success", ats.index)
	} else {
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Warnf("[%2d] failure", ats.index)
	}
	return
}

type TestContainer struct {
	CaseByte json.RawMessage
	Path     string
}

func (ats Suite) parseAndRunTest(v util.GenericJson, manifestDir, testFilePath string, k int, runParallel bool,
	r *report.ReportElement) bool {
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
		log.Error(fmt.Errorf("can not LoadManifestDataAsRawJson (%s): %s", testFilePath, err))
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
			waitCh <- true
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
					log.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
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
				log.Error(fmt.Errorf("can not render template (%s): %s", testFilePath, lErr))
				return false
			}

			//If the both objects are the same we did not have a template, but a mallformed json -> Call error
			if string(requestBytes) == string(testObj) {
				r.SaveToReportLog(err.Error())
				log.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
				return false
			}

			//We have a template -> One level deeper with rendered bytes
			return ats.parseAndRunTest([]byte(requestBytes), filepath.Join(manifestDir, dir),
				testFilePath, k, isParallelPathSpec, r)

		}
	}

	return true
}

func (ats Suite) runSingleTest(tc TestContainer, r *report.ReportElement, testFilePath string, loader template.Loader, k int, isParallel bool) (success bool) {
	r.SetName(testFilePath)

	var test Case
	jErr := cjson.Unmarshal(tc.CaseByte, &test)
	if jErr != nil {

		r.SaveToReportLog(jErr.Error())
		log.Error(fmt.Errorf("can not unmarshal single test (%s): %s", testFilePath, jErr))

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
	r.SetName(test.Name)
	success = test.runAPITestCase(r)

	if !success && !test.ContinueOnFailure {
		return false
	}

	return true
}

func (ats Suite) loadManifest() (res []byte, err error) {
	log.Tracef("Loading manifest: %s", ats.manifestPath)
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

func testGoRoutine(k, ki int, v json.RawMessage, ats Suite, testFilePath, manifestDir, dir string,
	r *report.ReportElement, loader template.Loader, waitCh, succCh chan bool, runParallel bool) {
	defer func() { <-waitCh }()
	var success bool

	//Check if is @ and if so load the test
	if util.IsPathSpec(v) {
		var sS string

		err := cjson.Unmarshal(v, &sS)
		if err != nil {
			r.SaveToReportLog(err.Error())
			log.Error(fmt.Errorf("can not unmarshal (%s): %s", testFilePath, err))
			succCh <- false
			return

		}
		success = ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), testFilePath, k+ki, runParallel, r)
	} else {
		success = ats.runSingleTest(TestContainer{CaseByte: v, Path: filepath.Join(manifestDir, dir)},
			r, testFilePath, loader, ki, runParallel)
	}

	succCh <- success

}
