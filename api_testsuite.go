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
	log.Infof("[%2d] '%s'", ats.index, ats.Name)

	r.NewChild(ats.Name)
	r.SetTestCount(len(ats.Tests))

	start := time.Now()
	success = ats.run()
	elapsed := time.Since(start)

	r.LeaveChild(success)
	if success {
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Infof("[%2d] success", ats.index)
	} else {
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Warnf("[%2d] failure", ats.index)
	}
	return
}

func (ats Suite) run() (success bool) {
	r := ats.reporter

	loader := template.NewLoader(ats.datastore)

	for k, v := range ats.Tests {

		tests, err := ats.loadTestCaseByte(v, ats.manifestDir, loader)
		if err != nil {
			r.SaveToReportLog(err.Error())
			log.Error(fmt.Errorf("can not loadTestCaseByte %s", err))
			return false
		}

		for _, testContainer := range tests {
			var test Case
			err := json.Unmarshal(testContainer.CaseByte, &test)
			if err != nil {
				r.SaveToReportLog(err.Error())
				log.Error(fmt.Errorf("can not unmarshal single test %s", err))
				return false
			}

			test.loader = loader
			test.manifestDir = filepath.Join(ats.manifestDir, testContainer.Path)
			test.reporter = r
			test.suiteIndex = ats.index
			test.index = k
			test.dataStore = ats.datastore
			test.ServerURL = ats.Config.ServerURL
			test.standardHeader = ats.StandardHeader
			test.standardHeaderFromStore = ats.StandardHeaderFromStore

			success := test.runAPITestCase()

			if !success && !test.ContinueOnFailure {
				return false
			}

		}
	}

	return true
}

type TestUnmarsh struct {
	CaseByte json.RawMessage
	Path     string
}

func (ats Suite) loadTestCaseByte(v util.GenericJson, manifestDir string, loader template.Loader) ([]TestUnmarsh, error) {
	rTests := make([]TestUnmarsh, 0)

	dir, testObj, err := template.LoadManifestDataAsRawJson(v, manifestDir)
	if err != nil {
		return rTests, err
	}

	var testCases []json.RawMessage
	err = cjson.Unmarshal(testObj, &testCases)

	if err != nil {
		err = nil

		rTests = make([]TestUnmarsh, 0)
		var singleTest json.RawMessage
		err = cjson.Unmarshal(testObj, &singleTest)
		if err == nil {
			rTests = append(rTests, TestUnmarsh{CaseByte: singleTest, Path: filepath.Join(manifestDir, dir)})
		} else {
			requestBytes, lErr := loader.Render(testObj, filepath.Join(manifestDir, dir), nil)
			if lErr != nil {
				return rTests, lErr
			}

			if string(requestBytes) == string(testObj) {
				return rTests, err
			}

			tests, llErr := ats.loadTestCaseByte(requestBytes, filepath.Join(manifestDir, dir), loader)
			if llErr != nil {
				return rTests, llErr
			}
			rTests = append(rTests, tests...)

			err = nil

		}
	} else {
		for _, v := range testCases {
			rTests = append(rTests, TestUnmarsh{CaseByte: v, Path: filepath.Join(manifestDir, dir)})
		}
	}

	tempTests := make([]TestUnmarsh, 0)

	for _, v := range rTests {

		if rune(v.CaseByte[1]) == rune('@') {
			var sS string
			cjson.Unmarshal(v.CaseByte, &sS)
			tests, err := ats.loadTestCaseByte(sS, v.Path, loader)
			if err != nil {
				return rTests, fmt.Errorf("could not load inner loadTestCaseByte: %s", err)
			}
			tempTests = append(tempTests, tests...)
		} else {
			requestBytes, err := loader.Render(v.CaseByte, v.Path, nil)
			if err != nil {
				return rTests, fmt.Errorf("could not render: %s", err)
			}
			v.CaseByte = requestBytes

			tempTests = append(tempTests, v)
		}
	}

	return tempTests, err
}

/*

func (ats Suite) unmarshalIntoTestCases(v util.GenericJson, loader template.Loader, manifestDir string)(rTests []TestUnmarsh, err error){
	rTests = make([]TestUnmarsh, 0)

	dir, testObj, err := template.LoadManifestDataAsRawJson(v, manifestDir)
	if err != nil {
		return rTests, err
	}

	testJson, err := json.Marshal(testObj)
	if err != nil {
		return rTests, err
	}

	var testCases []Case
	err = json.Unmarshal(testJson, &testCases)
	if err != nil {
		rTests = make([]TestUnmarsh,0)

		var singleTest Case
		err = json.Unmarshal(testJson, &singleTest)
		if err != nil {
			rTests = make([]TestUnmarsh,0)
			err = nil

				var genJson []util.GenericJson
				json.Unmarshal(testJson, &genJson)
				for _,iv := range genJson{
					iTests, err := ats.unmarshalIntoTestCases(iv,loader,filepath.Join(ats.manifestDir, dir))
					if err != nil {
						return rTests, err
					}
					rTests = append(rTests,iTests...)
				}


		}else {
			rTests = append(rTests, TestUnmarsh{Case:singleTest,Path:dir})
		}
	}else{
		for _,v := range testCases{
			rTests = append(rTests, TestUnmarsh{Case:v,Path:dir})
		}
	}

	return
}
*/
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
