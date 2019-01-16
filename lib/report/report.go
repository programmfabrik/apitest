package report

import (
	"fmt"
	"time"
)

type Report struct {
	root           *ReportElement
	currentElement *ReportElement
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
}

func NewReport() *Report {
	var report Report

	newElem := ReportElement{}
	newElem.SubTests = make([]*ReportElement, 0)
	newElem.StartTime = time.Now()

	report.root = &newElem
	report.currentElement = report.root

	return &report
}

//Navigate in tree functions
func (r *Report) NewChild(name string) {
	newElem := ReportElement{}
	newElem.SubTests = make([]*ReportElement, 0)

	newElem.Parent = r.currentElement
	newElem.Name = name
	newElem.StartTime = time.Now()

	r.currentElement.SubTests = append(r.currentElement.SubTests, &newElem)
	r.currentElement = &newElem
}

func (r *Report) LeaveChild(result bool) {
	if !result {
		r.currentElement.Failures++
	}
	r.goToParent()
}

func (r *Report) goToParent() {
	if r.currentElement.Parent == nil {
		return
	}

	r.currentElement.ExecutionTime = time.Since(r.currentElement.StartTime)
	r.currentElement = r.currentElement.Parent
}

//Report only from root on
func (r Report) GetTestResult(parsingFunction func(baseResult *ReportElement) []byte) []byte {
	return parsingFunction(r.root.getTestResult())
}

//Check if the testsuite did produce failures
func (r Report) DidFail() bool {
	if r.root.Failures > 0 {
		return true
	} else {
		return false
	}
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

func (r *Report) SetTestCount(counter int) {
	r.currentElement.TestCount = counter
}

func (r *Report) SaveToReportLog(v string) {
	r.currentElement.LogStorage = append(r.currentElement.LogStorage, v)
}

func (r *Report) SaveToReportLogF(v string, args ...interface{}) {
	r.currentElement.LogStorage = append(r.currentElement.LogStorage, fmt.Sprintf(v, args...))
}

func (r Report) GetLog() []string {
	return r.root.GetLog()
}
