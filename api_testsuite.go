package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/programmfabrik/fylr-apitest/lib/util"

	"github.com/programmfabrik/fylr-apitest/lib/api"
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
	datastore    *api.Datastore
	manifestDir  string
	manifestPath string
	reporter     *report.Report
	index        int
	serverURL    string
}

// NewTestSuite creates a new suite on which we execute our tests on
// Normally this only gets call from within the apitest main command
func NewTestSuite(
	config TestToolConfig,
	manifestPath string,
	r *report.Report,
	datastore *api.Datastore,
	index int,
) (suite Suite, err error) {
	suite = Suite{
		Config:       config,
		manifestDir:  filepath.Dir(manifestPath),
		manifestPath: manifestPath,
		reporter:     r,
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

	return suite, nil
}

func (ats Suite) Run() (success bool) {
	r := ats.reporter
	log.Infof("[%2d] '%s'", ats.index, ats.Name)

	r.NewChild(ats.Name)
	r.SetTestCount(len(ats.Tests))

	start := time.Now()

	success = true
	for k, v := range ats.Tests {
		sTestSuccess := ats.parseAndRunTest(v, ats.manifestDir, k)
		if !sTestSuccess {
			success = false
			break
		}
	}

	elapsed := time.Since(start)

	r.LeaveChild(success)
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

func (ats Suite) parseAndRunTest(v util.GenericJson, manifestDir string, k int) bool {
	//Init variables
	r := ats.reporter
	loader := template.NewLoader(ats.datastore)

	//Get the Manifest with @ logic
	dir, testObj, err := template.LoadManifestDataAsRawJson(v, manifestDir)
	if err != nil {
		r.SaveToReportLog(err.Error())
		log.Error(fmt.Errorf("can not LoadManifestDataAsRawJson: %s", err))
		return false
	}

	//Try to directly unmarshal the manifest into testcase array
	var testCases []json.RawMessage
	err = cjson.Unmarshal(testObj, &testCases)
	if err == nil {
		//Did work -> No go template

		var success bool
		//Run single tests
		for ki, v := range testCases {

			//Check if is @ and if so load the test
			if rune(v[1]) == rune('@') {
				var sS string

				err := cjson.Unmarshal(v, &sS)
				if err != nil {
					r.SaveToReportLog(err.Error())
					log.Error(fmt.Errorf("can not unmarshal: %s", err))
					return false
				}

				success = ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), k+ki)
			} else {
				success = ats.runSingleTest(TestContainer{CaseByte: v, Path: filepath.Join(manifestDir, dir)}, r, loader, ki)
			}

			if !success {
				return false
			}
		}
	} else {
		//We were not able unmarshal into array, so we try to unmarshal into raw message
		var singleTest json.RawMessage
		err = cjson.Unmarshal(testObj, &singleTest)
		if err == nil {
			//Did work to unmarshal -> no go template

			//Check if is @ and if so load the test
			if rune(testObj[1]) == rune('@') {
				var sS string

				err := cjson.Unmarshal(testObj, &sS)
				if err != nil {
					r.SaveToReportLog(err.Error())
					log.Error(fmt.Errorf("can not unmarshal: %s", err))
					return false
				}

				return ats.parseAndRunTest(sS, filepath.Join(manifestDir, dir), k)
			} else {
				return ats.runSingleTest(TestContainer{CaseByte: testObj, Path: filepath.Join(manifestDir, dir)}, r, loader, k)
			}
		} else {
			//Did not work -> Could be go template or a mallformed json
			requestBytes, lErr := loader.Render(testObj, filepath.Join(manifestDir, dir), nil)
			if lErr != nil {
				return false
			}

			//If the both objects are the same we did not have a template, but a mallformed json -> Call error
			if string(requestBytes) == string(testObj) {
				r.SaveToReportLog(err.Error())
				log.Error(fmt.Errorf("can not unmarshal json: %s", err))
				return false
			}

			//We have a template -> One level deeper with rendered bytes
			return ats.parseAndRunTest([]byte(requestBytes), filepath.Join(manifestDir, dir), k)

		}
	}

	return true
}

func (ats Suite) runSingleTest(tc TestContainer, r *report.Report, loader template.Loader, k int) (success bool) {
	var test Case
	jErr := cjson.Unmarshal(tc.CaseByte, &test)
	if jErr != nil {
		r.SaveToReportLog(jErr.Error())
		log.Error(fmt.Errorf("can not unmarshal single test %s", jErr))
		return false
	}

	test.loader = loader
	test.manifestDir = tc.Path
	test.reporter = r
	test.suiteIndex = ats.index
	test.index = k
	test.dataStore = ats.datastore
	test.ServerURL = ats.Config.ServerURL
	test.standardHeader = ats.StandardHeader
	test.standardHeaderFromStore = ats.StandardHeaderFromStore

	success = test.runAPITestCase()

	if !success && !test.ContinueOnFailure {
		return false
	}

	return true
}

func (ats Suite) loadManifest() (res []byte, err error) {
	loader := template.NewLoader(ats.datastore)
	manifestFile, err := filesystem.Fs.Open(ats.manifestPath)
	if err != nil {
		return res, fmt.Errorf("error opening manifestPath: %s", err)
	}
	defer manifestFile.Close()

	manifestTmpl, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return res, fmt.Errorf("error loading manifest: %s", err)
	}
	return loader.Render(manifestTmpl, ats.manifestDir, nil)
}
