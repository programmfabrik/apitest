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
	"github.com/programmfabrik/fylr-apitest/lib/logging"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"github.com/programmfabrik/fylr-apitest/lib/template"
)

// Suite defines the structure of our apitest
// We do read this in with the config loader
type Suite struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Tests        []util.GenericJson     `json:"tests"`
	RequirePaths []string               `json:"require"`
	Store        map[string]interface{} `json:"store"`

	Config              TestToolConfig
	datastore           *api.Datastore
	manifestDir         string
	manifestPath        string
	reporter            *report.Report
	index               int
	executeRequirements bool
	serverURL           string
}

// NewTestSuite creates a new suite on which we execute our tests on
// Normally this only gets call from within the apitest main command
func NewTestSuite(
	config TestToolConfig,
	manifestPath string,
	r *report.Report,
	executeRequirements bool,
	datastore *api.Datastore,
	index int,
) (suite Suite, err error) {
	suite = Suite{
		Config:              config,
		manifestDir:         filepath.Dir(manifestPath),
		manifestPath:        manifestPath,
		reporter:            r,
		executeRequirements: executeRequirements,
		datastore:           datastore,
		index:               index,
	}

	manifest, err := suite.loadManifest()
	if err != nil {
		return suite, fmt.Errorf("error loading manifest: %s", err)
	}

	if err = cjson.Unmarshal(manifest, &suite); err != nil {
		return suite, fmt.Errorf("error unmarshaling manifest '%s': %s", manifestPath, err)
	}

	// init store
	err = suite.datastore.SetMap(suite.Store)
	if err != nil {
		err = fmt.Errorf("error setting datastore map:%s", err)
	}

	return suite, nil
}

func (ats Suite) Run() (success bool) {
	r := ats.reporter
	logging.Infof("[%2d] '%s'", ats.index, ats.Name)

	r.NewChild(ats.Name)
	r.SetTestCount(len(ats.Tests))

	start := time.Now()
	success = ats.run()
	elapsed := time.Since(start)

	r.LeaveChild(success)
	if success {
		logging.InfoWithFieldsF(map[string]interface{}{"elapsed": elapsed.Seconds()}, "[%2d] success", ats.index)
	} else {
		logging.WarnWithFieldsF(map[string]interface{}{"elapsed": elapsed.Seconds()}, "[%2d] failure", ats.index)
	}
	return
}

func (ats Suite) run() (success bool) {
	if ats.executeRequirements {
		success = ats.runRequirements()
		if !success {
			return false
		}
	}

	return ats.runTestCases()
}

func (ats Suite) runRequirements() (success bool) {
	//new empty reporter to don't include requirements in Testreport
	r := report.NewReport()

	if ats.RequirePaths == nil {
		return true
	}

	logging.Infof("[%2d] %s", ats.index, "run requirements")
	for _, parentPath := range ats.RequirePaths {
		suite, err := NewTestSuite(
			ats.Config,
			filepath.Join(ats.manifestDir, parentPath),
			r,
			ats.executeRequirements,
			ats.datastore,
			ats.index+1,
		)
		if err != nil {
			r.SaveToReportLog(err.Error())
			logging.Errorf("[%2d] error loading parent suite: %s", ats.index, err)
			return false
		}

		pSuccess := suite.Run()
		if !pSuccess {
			logging.Warnf("[%2d] requirements: failure", ats.index)
			return false
		}
	}

	logging.Infof("[%2d] requirements: success", ats.index)
	return true
}

/*
Runs TestCases of the TestSuite
We have to create the session at this point, otherwise
it might get deleted by a previous post to session/purgeall.
*/
func (ats Suite) runTestCases() (success bool) {
	/*defer func() {
		if err := recover(); err != nil {
			logging.Error(err)
			success = false
		}
	}()*/

	r := ats.reporter
	//datastoreShare := api.NewStoreShare(ats.datastore)

	loader := template.NewLoader(ats.datastore)

	for k, v := range ats.Tests {
		tests := make([]Case, 0)

		intFilepath, testObj, err := template.LoadManifestDataAsObject(v, ats.manifestDir, loader)
		if err != nil {
			r.SaveToReportLog(err.Error())
			logging.Error(err)
			return false
		}

		testJson, err := json.Marshal(testObj)
		if err != nil {
			r.SaveToReportLog(err.Error())
			logging.Error(err)
			return false
		}

		err = json.Unmarshal(testJson, &tests)
		if err != nil {
			singleTest := Case{}
			err = json.Unmarshal(testJson, &singleTest)
			if err != nil {
				r.SaveToReportLog(err.Error())
				logging.Error(err)
				return false
			}

			tests = append(tests, singleTest)
		}

		for _, test := range tests {
			test.loader = loader
			test.manifestDir = filepath.Join(ats.manifestDir, intFilepath)
			test.reporter = r
			test.suiteIndex = ats.index
			test.index = k
			test.dataStore = ats.datastore
			test.ServerURL = ats.Config.ServerURL

			success := test.runAPITestCase()

			if !success && !test.ContinueOnFailure {
				return false
			}
		}
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
