// Copyright Programmfabrik GmbH
// All Rights Reserved
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/report"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reportFormat, reportFile, serverURL, httpServerReplaceHost                                     string
	keepRunning, logNetwork, logDatastore, logVerbose, logTimeStamp, logShort, logCurl, stopOnFail bool
	rootDirectorys, singleTests                                                                    []string
	limitRequest, limitResponse, reportStatsGroups                                                 uint
	// set via -ldflags during build
	buildCommit, buildTime, buildVersion string
)

func init() {
	testCMD.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./apitest.yml", "config file")

	testCMD.PersistentFlags().StringVar(
		&serverURL, "server", "",
		"URL of the Server. Overwrites server URL in yml config.")

	testCMD.PersistentFlags().StringVar(
		&httpServerReplaceHost, "replace-host", "",
		"HTTP Server replacement host to be used in replace_host template function.")

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

	testCMD.PersistentFlags().BoolVarP(
		&logShort, "log-short", "", false,
		"short log on success into console")

	testCMD.PersistentFlags().StringVar(
		&reportFile, "report-file", "",
		"Defines where the log statements should be saved.")

	testCMD.PersistentFlags().StringVar(
		&reportFormat, "report-format", "",
		"Defines how the report statements should be saved. [junit/json/stats]")

	testCMD.PersistentFlags().UintVarP(
		&reportStatsGroups, "report-format-stats-group", "", 4,
		"Create report format stats groups distribution (default 4)")

	testCMD.PersistentFlags().UintVarP(
		&limitRequest, "limit-request", "", 20,
		"Limit the lines of request log output to n lines (set to 0 for no limit)")
	testCMD.PersistentFlags().UintVarP(
		&limitResponse, "limit-response", "", 20,
		"Limit the lines of response log output to n lines (set to 0 for no limit)")

	testCMD.PersistentFlags().BoolVar(
		&logCurl, "curl-bash", false,
		"Log network output as bash curl command")

	testCMD.PersistentFlags().BoolVar(
		&keepRunning, "keep-running", false,
		"Before returning from each test suite, prompt for a keyboard interrupt")

	testCMD.PersistentFlags().BoolVar(
		&stopOnFail, "stop-on-fail", false,
		"Stop execution of later test suites if a test suite fails")

	// Bind the flags to overwrite the yml config if they are set
	viper.BindPFlag("apitest.report.file", testCMD.PersistentFlags().Lookup("report-file"))
	viper.BindPFlag("apitest.report.format", testCMD.PersistentFlags().Lookup("report-format"))
	viper.BindPFlag("apitest.server", testCMD.PersistentFlags().Lookup("server"))
	viper.BindPFlag("apitest.log.short", testCMD.PersistentFlags().Lookup("log-short"))
	viper.BindPFlag("apitest.limit.request", testCMD.PersistentFlags().Lookup("limit-request"))
	viper.BindPFlag("apitest.limit.response", testCMD.PersistentFlags().Lookup("limit-response"))

	// println("The latest apitest tool, v 68")
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

	// timestamp: start of all tests
	start := time.Now()

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

	currDir, _ := os.Getwd()
	absPath := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(currDir, p)
	}

	server := Config.Apitest.Server
	reportFormat = Config.Apitest.Report.Format
	if Config.Apitest.Report.File != "" {
		reportFile = absPath(Config.Apitest.Report.File)
	}

	rep := report.NewReport()
	rep.StatsGroups = int(reportStatsGroups)
	rep.Version = buildCommit

	// Save the config into TestToolConfig
	testToolConfig, err := NewTestToolConfig(server, rootDirectorys, logNetwork, logVerbose, Config.Apitest.Log.Short)
	if err != nil {
		logrus.Error(err)
		if reportFile != "" {
			rep.WriteToFile(reportFile, reportFormat)
		}
	}

	// Actually run the tests
	// Run test function
	runSingleTest := func(manifestPath string, manifestDir string, reportElem *report.ReportElement) (success bool) {
		store := datastore.NewStore(logVerbose || logDatastore)
		for k, v := range Config.Apitest.StoreInit {
			err := store.Set(k, v)
			if err != nil {
				logrus.Errorf("Could not add init value for datastore Key: '%s', Value: '%v'. %s", k, v, err)
			}
		}

		suite, err := NewTestSuite(testToolConfig, manifestPath, manifestDir, reportElem, store, 0)
		if err != nil {
			logrus.Error(err)
			if reportFile != "" {
				rep.WriteToFile(reportFile, reportFormat)
			}
			return false
		}

		return suite.Run()
	}

	// Decide if run only one test
	if len(singleTests) > 0 {
		for _, singleTest := range singleTests {
			c := rep.Root().NewChild(singleTest)

			success := runSingleTest(absPath(singleTest), singleTest, c)
			c.Leave(success)

			if stopOnFail && !success {
				break
			}
		}
	} else {
		for _, singlerootDirectory := range testToolConfig.TestDirectories {
			manifestPath := filepath.Join(singlerootDirectory, "manifest.json")
			c := rep.Root().NewChild(manifestPath)

			success := runSingleTest(absPath(manifestPath), manifestPath, c)
			c.Leave(success)

			if stopOnFail && !success {
				break
			}
		}
	}

	if reportFile != "" {
		rep.WriteToFile(reportFile, reportFormat)
	}

	// timestamp: end of all tests
	end := time.Now()

	time_format := "2006-01-02 15:04:05"
	format_start := start.Format(time_format)
	format_end := end.Format(time_format)
	format_runtime := time.Since(start).String()
	logrus.Infof("All done in %s. Start: %s. End: %s.", format_runtime, format_start, format_end)

	if rep.DidFail() {
		os.Exit(1)
	}
}
