package report

import (
	"encoding/xml"
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"strconv"
	"strings"
	"time"
)

type XMLRoot struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Id         string      `xml:"id,attr"`
	Name       string      `xml:"name,attr"`
	Failures   int         `xml:"failures,attr"`
	Time       float64     `xml:"time,attr"`
	Tests      int         `xml:"tests,attr"`
	Testsuites []testsuite `xml:"testsuite"`
}

type testsuite struct {
	Id        string     `xml:"id,attr"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Time      float64    `xml:"time,attr"`
	Testcases []testcase `xml:"testcase"`
}

type testcase struct {
	Id      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
	Time    float64  `xml:"time,attr"`
	Failure *failure `xml:"failure,omitempty"`
}

type failure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
}

type JUnitReporter struct {
	report Report
}

//Print the result to the console
func ParseJSONResult(baseResult *ReportElement) []byte {
	jsonResult, _ := cjson.MarshalIndent(baseResult, "", " ")

	return jsonResult
}

//Print the result to the console
func ParseJUnitResult(baseResult *ReportElement) []byte {

	testName := time.Now().Format("2006-01-02 15:04")
	result := XMLRoot{
		Name:     testName,
		Id:       testName,
		Failures: baseResult.Failures,
		Tests:    baseResult.TestCount,
		Time:     baseResult.ExecutionTime.Seconds(),
	}

	for k, v := range baseResult.SubTests {
		newTestSuite := testsuite{
			Id:       strconv.Itoa(k),
			Time:     v.ExecutionTime.Seconds(),
			Failures: v.Failures,
			Tests:    v.TestCount,
			Name:     strings.Replace(v.Name, ".", ":", -1),
		}

		flattenSubTests := v.SubTests.Flat()

		for ik, iv := range flattenSubTests {
			newTestCase := testcase{
				Id:   strconv.Itoa(ik),
				Time: iv.ExecutionTime.Seconds(),
				Name: strings.Replace(iv.Name, ".", ":", -1),
			}

			if iv.Failures > 0 {
				newTestCase.Failure = &failure{Type: "ERROR"}

				for _, jv := range iv.GetLog() {
					newTestCase.Failure.Message = fmt.Sprintf("%s\n\n%s", newTestCase.Failure.Message, jv)
				}
			}

			newTestSuite.Testcases = append(newTestSuite.Testcases, newTestCase)
		}
		result.Testsuites = append(result.Testsuites, newTestSuite)
	}

	xmlResult, _ := xml.MarshalIndent(result, "", "\t")

	return xmlResult
}
