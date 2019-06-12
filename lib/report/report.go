package report

import (
	"fmt"
	"sync"
	"time"
)

type Report struct {
	root *ReportElement
	m    *sync.Mutex
}

func (r Report) Root() *ReportElement {
	return r.root
}

func NewReport() *Report {
	var report Report

	newElem := ReportElement{}
	newElem.SubTests = make([]*ReportElement, 0)
	newElem.StartTime = time.Now()
	newElem.report = &report
	newElem.m = &sync.Mutex{}

	report.root = &newElem
	report.m = &sync.Mutex{}

	return &report
}

//GetTestResult Parses the test report with the given function from the report root on
func (r Report) GetTestResult(parsingFunction func(baseResult *ReportElement) []byte) []byte {

	return parsingFunction(r.root.getTestResult())
}

//DidFail, Check if the testsuite did produce failures
func (r Report) DidFail() bool {

	if r.root.Failures > 0 {
		return true
	} else {
		return false
	}
}

func (r Report) GetLog() []string {
	return r.root.GetLog()
}

type ReportElement struct {
	Failures      int              `json:"failures"`
	TestCount     int              `json:"test_count,omitempty"`
	ExecutionTime time.Duration    `json:"execution_time_ns"`
	StartTime     time.Time        `json:"-"`
	Name          string           `json:"name,omitempty"`
	LogStorage    []string         `json:"log,omitempty"`
	SubTests      []*ReportElement `json:"sub_tests,omitempty"`
	Parent        *ReportElement   `json:"-"`
	report        *Report
	m             *sync.Mutex
}

//NewChild create new report element and return its reference
func (r *ReportElement) NewChild(name string) (newElem *ReportElement) {
	r.report.m.Lock()
	defer r.report.m.Unlock()

	newElem = &ReportElement{}
	newElem.SubTests = make([]*ReportElement, 0)
	newElem.m = &sync.Mutex{}

	newElem.Parent = r
	newElem.Name = name
	newElem.StartTime = time.Now()
	newElem.report = r.report

	r.SubTests = append(r.SubTests, newElem)

	return
}

func (r *ReportElement) SetName(name string) {
	r.m.Lock()
	defer r.m.Unlock()

	r.Name = name
}

func (r *ReportElement) Leave(result bool) {
	r.m.Lock()
	defer r.m.Unlock()

	if !result {
		r.Failures++
	}
	r.ExecutionTime = time.Since(r.StartTime)
}

//aggregate results of subtests
func (r *ReportElement) getTestResult() *ReportElement {
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

func (r ReportElement) GetLog() []string {
	errors := make([]string, 0)

	//root Errors
	for _, singleMessage := range r.LogStorage {
		errors = append(errors, singleMessage)
	}

	//Child errors
	for _, singleTest := range r.SubTests {
		errors = append(errors, singleTest.GetLog()...)
	}
	return errors
}

func (r *ReportElement) SetTestCount(counter int) {
	r.m.Lock()
	defer r.m.Unlock()

	r.TestCount = counter
}

func (r *ReportElement) SaveToReportLog(v string) {
	r.m.Lock()
	defer r.m.Unlock()

	r.LogStorage = append(r.LogStorage, v)
}

func (r *ReportElement) SaveToReportLogF(v string, args ...interface{}) {
	r.m.Lock()
	defer r.m.Unlock()

	r.LogStorage = append(r.LogStorage, fmt.Sprintf(v, args...))
}
