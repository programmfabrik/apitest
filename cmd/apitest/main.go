// Copyright Programmfabrik GmbH
// All Rights Reserved
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/report"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reportFormat, reportFile, serverURL                                            string
	logNetwork, logDatastore, logVerbose, logTimeStamp, limit, logCurl, stopOnFail bool
	rootDirectorys, singleTests                                                    []string
)

func init() {
	testCMD.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./apitest.yml", "config file")

	testCMD.PersistentFlags().StringVar(
		&serverURL, "server", "",
		"URL of the Server. Overwrites server URL in yml config.")

	testCMD.PersistentFlags().StringSliceVarP(
		&rootDirectorys, "directory", "d", []string{"."},
		"path to directory containing the tests.")

	testCMD.PersistentFlags().StringSliceVarP(
		&singleTests, "single", "s", []string{},
		"path to a single manifest. Runs only that specified testsuite")

	testCMD.PersistentFlags().BoolVarP(
		&logNetwork, "log-network", "n", false,
		"log all network traffic to console")
	testCMD.PersistentFlags().BoolVarP(
		&logVerbose, "log-verbose", "v", false,
		"log datastore operations and information about repeating request to console")
	testCMD.PersistentFlags().BoolVar(
		&logDatastore, "log-datastore", false,
		"log datastore operations")

	testCMD.PersistentFlags().BoolVarP(
		&logTimeStamp, "log-timestamp", "t", false,
		"log full timestamp into console")

	testCMD.PersistentFlags().StringVar(
		&reportFile, "report-file", "",
		"Defines where the log statements should be saved.")

	testCMD.PersistentFlags().StringVar(
		&reportFormat, "report-format", "",
		"Defines how the report statements should be saved. [junit/json]")

	testCMD.PersistentFlags().BoolVarP(
		&limit, "limit", "l", false,
		"Limit the lines of request log output. Set limits in apitest.yml")

	testCMD.PersistentFlags().BoolVar(
		&logCurl, "curl-bash", false,
		"Log network output as bash curl command")

	testCMD.PersistentFlags().BoolVar(
		&stopOnFail, "stop-on-fail", false,
		"Stop execution of later test suites if a test suite fails")

	// Bind the flags to overwrite the yml config if they are set
	viper.BindPFlag("apitest.report.file", testCMD.PersistentFlags().Lookup("report-file"))
	viper.BindPFlag("apitest.report.format", testCMD.PersistentFlags().Lookup("report-format"))
	viper.BindPFlag("apitest.server", testCMD.PersistentFlags().Lookup("server"))
}

var testCMD = &cobra.Command{
	Args:             cobra.MaximumNArgs(0),
	PersistentPreRun: setup,
	Use:              "apitest",
	Short:            "Apitester lets you define API tests on the go",
	Long:             "A fast and flexible API testing tool. Helping you to define API tests on the go",
	Run:              runApiTests,
}

func main() {
	err := testCMD.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var cfgFile string

func setup(ccmd *cobra.Command, args []string) {
	// Load yml config
	LoadConfig(cfgFile)

	// Set log verbosity to trace
	logrus.SetLevel(logrus.TraceLevel)

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: logTimeStamp,
	})
}

func runApiTests(cmd *cobra.Command, args []string) {

	// Check if paths are valid
	for _, rootDirectory := range rootDirectorys {
		if _, err := os.Stat(rootDirectory); rootDirectory != "." && os.IsNotExist(err) {
			logrus.Fatalf("The path '%s' for the test folders is not valid", rootDirectory)
		}
	}
	for _, singleTest := range singleTests {
		if _, err := os.Stat(singleTest); singleTest != "" && os.IsNotExist(err) {
			logrus.Fatalf("The path '%s' for the single test is not valid", singleTest)
		}
	}

	server := Config.Apitest.Server
	reportFormat = Config.Apitest.Report.Format
	reportFile = Config.Apitest.Report.File

	// Save the config into TestToolConfig
	testToolConfig, err := NewTestToolConfig(server, rootDirectorys, logNetwork, logVerbose)
	if err != nil {
		logrus.Fatal(err)
	}

	// Actually run the tests
	// Run test function
	runSingleTest := func(manifestPath string, r *report.ReportElement) (success bool) {
		store := datastore.NewStore(logVerbose || logDatastore)
		for k, v := range Config.Apitest.StoreInit {
			err := store.Set(k, v)
			if err != nil {
				logrus.Errorf("Could not add init value for datastore Key: '%s', Value: '%v'. %s", k, v, err)
			}
		}

		suite, err := NewTestSuite(testToolConfig, manifestPath, r, store, 0)
		if err != nil {
			logrus.Fatal(err)
		}

		return suite.Run()
	}

	r := report.NewReport()

	// Decide if run only one test
	if len(singleTests) > 0 {
		for _, singleTest := range singleTests {
			absManifestPath, _ := filepath.Abs(singleTest)
			c := r.Root().NewChild(singleTest)

			success := runSingleTest(absManifestPath, c)
			c.Leave(success)
			if stopOnFail && !success {
				break
			}
		}
	} else {
		for _, singlerootDirectory := range testToolConfig.TestDirectories {
			manifestPath := filepath.Join(singlerootDirectory, "manifest.json")
			absManifestPath, _ := filepath.Abs(manifestPath)
			c := r.Root().NewChild(manifestPath)

			success := runSingleTest(absManifestPath, c)
			c.Leave(success)
			if stopOnFail && !success {
				break
			}
		}
	}

	// Create report
	if reportFile != "" {
		var parsingFunction func(baseResult *report.ReportElement) []byte
		switch reportFormat {
		case "junit":
			parsingFunction = report.ParseJUnitResult
		case "json":
			parsingFunction = report.ParseJSONResult
		default:
			logrus.Errorf(
				"Given report format '%s' not supported. Saving report '%s' as json",
				reportFormat,
				reportFile)

			parsingFunction = report.ParseJSONResult
		}

		err = ioutil.WriteFile(reportFile, r.GetTestResult(parsingFunction), 0644)
		if err != nil {
			logrus.Errorf("Could not save report into file: %s", err)
		}
	}

	if r.DidFail() {
		os.Exit(1)
	}
}
