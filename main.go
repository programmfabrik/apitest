// Copyright Programmfabrik GmbH
// All Rights Reserved
package main

import (
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/api"

	"github.com/programmfabrik/fylr-apitest/lib/logging"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"os"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

)

var (
	reportFormat, reportFile string
	verbosity int
	noRequirements bool
	rootDirectorys, singleTests []string
)


func init() {
	//Configure all the flags that fylr-apitest offers
	TestCMD.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./fylr.yml", "config file")
	TestCMD.PersistentFlags().String("log-console-level", "info", "console loglevel")
	TestCMD.PersistentFlags().Bool("log-console-enable", true, "Set to true to enable console logging")
	TestCMD.PersistentFlags().StringSliceVarP(
		&rootDirectorys, "directory", "d", []string{"."},
		"path to directory containing the tests.")

	TestCMD.PersistentFlags().Bool(
		"no-requirements", false,
		"don't run requirements for the testsuite.")

	TestCMD.PersistentFlags().StringSliceVarP(
		&singleTests, "single", "s", []string{},
		"path to a single manifest. Runs only that specified testsuite")

	TestCMD.PersistentFlags().IntVarP(
		&verbosity, "verbosity", "v", -1,
		`in [-1, 0, 1, 2], defines logging of requests and responses of the programm
-1: dont log requests or responses
0: log requests and responses in case of test failure
1: log requests and responses that are documented in the manifest
2: log all requests and responses
verbosities >= 0 automatically set the global loglevel to debug`)

	TestCMD.PersistentFlags().StringVar(
		&reportFile, "report-file", "",
		"Defines where the log statements should be saved.")

	TestCMD.PersistentFlags().StringVar(
		&reportFormat, "report-format", "",
		"Defines how the report statements should be saved. [junit/json]")


	//Bind the flags to overwrite the yml config if they are set
	viper.BindPFlag("log.console.enable", TestCMD.PersistentFlags().Lookup("log-console-enable"))
	viper.BindPFlag("log.console.level", TestCMD.PersistentFlags().Lookup("log-console-level"))
	viper.BindPFlag("apitest.report.file", TestCMD.PersistentFlags().Lookup("report-file"))
	viper.BindPFlag("apitest.report.format", TestCMD.PersistentFlags().Lookup("report-format"))
}


var TestCMD = &cobra.Command{
	Args:             cobra.MaximumNArgs(0),
	PersistentPreRun: setup,
	Use:   "fylr apitest",
	Short: "flyr Apitester lets you define API tests on the go",
	Long:  `A fast and flexible API testing tool. Helping you to define API tests on the go`,
	Run: func(cmd *cobra.Command, args []string) {

		noRequirements = (cmd.Flag("no-requirements").Value.String() == "true")

		//Check if paths are valid

		for _, rootDirectory := range rootDirectorys {
			if _, err := os.Stat(rootDirectory); rootDirectory != "." && os.IsNotExist(err) {
				logging.Errorf("The path '%s' for the test folders is not valid", rootDirectory)
				os.Exit(1)
			}
		}
		for _, singleTest := range singleTests {
			if _, err := os.Stat(singleTest); singleTest != "" && os.IsNotExist(err) {
				logging.Errorf("The path '%s' for the single test is not valid", singleTest)
				os.Exit(1)
			}
		}

		if err := logging.InitApiTestLogging(verbosity); err != nil {
			logging.Errorf("Could not configure verbosity of apitest logging")
			os.Exit(1)
		}

		serverUrl := FylrConfig.Apitest.Server
		dbName := FylrConfig.Apitest.DBName
		reportFormat = FylrConfig.Apitest.Report.Format
		reportFile = FylrConfig.Apitest.Report.File

		//Save the config into TestToolConfig
		testToolConfig, err := NewTestToolConfig(serverUrl, dbName, rootDirectorys)
		if err != nil {
			logging.Error(err)
			os.Exit(1)
		}

		datastore := api.NewStore()
		for k, v := range FylrConfig.Apitest.StoreInit {
			datastore.Set(k, v)
			logging.Infof("Add Init value for datastore Key: '%s', Value: '%v'", k, v)
		}

		//Actually run the tests
		//Run test function
		runSingleTest := func(manifestPath string, r *report.Report) {
			suite, err := NewTestSuite(
				&http.Client{},
				testToolConfig,
				manifestPath,
				r,
				!noRequirements,
				datastore,
				0,
			)
			if err != nil {
				logging.Error(err)
				os.Exit(1)
			}

			suite.Run()
		}

		r := report.NewReport()

		//Decide if run only one test
		if len(singleTests) > 0 {
			for _, singleTest := range singleTests {
				absManifestPath, _ := filepath.Abs(singleTest)
				runSingleTest(absManifestPath, r)
			}
		} else {
			for _, singlerootDirectory := range testToolConfig.TestDirectories {
				manifestPath := filepath.Join(singlerootDirectory, "manifest.json")
				absManifestPath, _ := filepath.Abs(manifestPath)
				runSingleTest(absManifestPath, r)
			}
		}

		//Create report
		if reportFile != "" {
			var parsingFunction func(baseResult *report.ReportElement) []byte
			switch reportFormat {
			case "junit":
				parsingFunction = report.ParseJUnitResult
			case "json":
				parsingFunction = report.ParseJSONResult
			default:
				logging.Errorf(
					"Given report format '%s' not supported. Saving report '%s' as json",
					reportFormat,
					reportFile)

				parsingFunction = report.ParseJSONResult
			}

			err = ioutil.WriteFile(reportFile, r.GetTestResult(parsingFunction), 0644)
			if err != nil {
				fmt.Println("Could not save into file: ", err)
			}
		}

		if r.DidFail() {
			os.Exit(1)
		}
	},
}


func main() {
	err := TestCMD.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}


var cfgFile string

func setup(ccmd *cobra.Command, args []string) {
	//Load yml config
	LoadConfig(cfgFile)

	//Setup logging
	if err := logging.ConfigureLogging(
		viper.GetBool("log.console.enable"),
		viper.GetString("log.console.level"));	err != nil{
		log.Fatalf("error configuring logging: %s", err)
	}
}

