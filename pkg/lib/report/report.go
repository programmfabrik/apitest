package report

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Report struct {
	root *Element
	m    *sync.Mutex
}

func (r Report) Root() *Element {
	return r.root
}

func NewReport() *Report {
	var report Report

	newElem := Element{}
	newElem.SubTests = []*Element{}
	newElem.StartTime = time.Now()
	newElem.report = &report
	newElem.m = &sync.Mutex{}

	report.root = &newElem
	report.m = &sync.Mutex{}

	return &report
}

// GetTestResult Parses the test report with the given function from the report root on
func (r Report) GetTestResult(parsingFunction func(baseResult *Element) []byte) []byte {

	return parsingFunction(r.root.getTestResult())
}

// DidFail Check if the testsuite did produce failures
func (r Report) DidFail() bool {

	if r.root.Failures > 0 {
		return true
	}
	return false
}

func (r Report) GetLog() []string {
	return r.root.GetLog()
}

type Element struct {
	Failures      int           `json:"failures"`
	TestCount     int           `json:"test_count,omitempty"`
	ExecutionTime time.Duration `json:"execution_time_ns"`
	StartTime     time.Time     `json:"-"`
	Name          string        `json:"name,omitempty"`
	LogStorage    []string      `json:"log,omitempty"`
	SubTests      Elements      `json:"sub_tests,omitempty"`
	Parent        *Element      `json:"-"`
	NoLogTime     bool          `json:"-"`
	Failure       string        `json:"failure,omitempty"`
	report        *Report
	m             *sync.Mutex
}

type Elements []*Element

func (re Elements) Flat() Elements {
	rElements := Elements{}
	for _, v := range re {
		rElements = append(rElements, v)

		if len(v.SubTests) != 0 {
			rElements = append(rElements, v.SubTests.Flat()...)
		}
	}
	return rElements
}

// NewChild create new report element and return its reference
func (r *Element) NewChild(name string) (newElem *Element) {
	r.report.m.Lock()
	defer r.report.m.Unlock()

	name = strings.Replace(name, ".", "_", -1)

	newElem = &Element{}
	newElem.SubTests = []*Element{}
	newElem.m = &sync.Mutex{}

	newElem.Parent = r
	newElem.NoLogTime = r.NoLogTime
	newElem.Name = name
	newElem.StartTime = time.Now()
	newElem.report = r.report

	r.SubTests = append(r.SubTests, newElem)

	return
}

func (r *Element) SetName(name string) {
	r.m.Lock()
	defer r.m.Unlock()
	name = strings.Replace(name, ".", "_", -1)
	r.Name = name
}

func (r *Element) Leave(result bool) {
	r.m.Lock()
	defer r.m.Unlock()

	if len(r.SubTests) == 0 {
		r.TestCount++
		if !result {
			r.Failures++
		}
	}
	r.ExecutionTime = time.Since(r.StartTime)
}

// aggregate results of subtests
func (r *Element) getTestResult() *Element {
	for _, v := range r.SubTests {
		subResults := v.getTestResult()
		r.TestCount += subResults.TestCount
		r.Failures += subResults.Failures
	}

	if r.ExecutionTime == 0 {
		r.ExecutionTime = time.Since(r.StartTime)
	}

	return r
}

func (r Element) GetLog() []string {
	var errors []string

	// root Errors
	for _, singleMessage := range r.LogStorage {
		errors = append(errors, singleMessage)
	}

	// Child errors
	for _, singleTest := range r.SubTests {
		errors = append(errors, singleTest.GetLog()...)
	}
	return errors
}

func (r *Element) SaveToReportLog(v string) {
	r.m.Lock()
	defer r.m.Unlock()

	if r.NoLogTime {

		r.LogStorage = append(r.LogStorage, v)
	} else {
		r.LogStorage = append(r.LogStorage, fmt.Sprintf("[%s] %s", time.Now().Format("02.01.2006 15:04:05.000 -0700"), v))

	}
}

func (r *Element) SaveToReportLogF(v string, args ...interface{}) {
	r.m.Lock()
	defer r.m.Unlock()

	r.LogStorage = append(r.LogStorage, fmt.Sprintf(v, args...))
}

// WriteToFile write the report into the report file
func (r *Report) WriteToFile(reportFile, reportFormat string) error {
	var parsingFunction func(baseResult *Element) []byte
	switch reportFormat {
	case "junit":
		parsingFunction = ParseJUnitResult
	case "json":
		parsingFunction = ParseJSONResult
	default:
		logrus.Errorf(
			"Given report format '%s' not supported. Saving report '%s' as json",
			reportFormat,
			reportFile)

		parsingFunction = ParseJSONResult
	}

	err := ioutil.WriteFile(reportFile, r.GetTestResult(parsingFunction), 0644)
	if err != nil {
		logrus.Errorf("Could not save report into file: %s", err)
		return err
	}

	return nil
}
