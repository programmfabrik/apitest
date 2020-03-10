package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type XMLRoot struct {
	XMLName    xml.Name    `xml:"testsuites"`
	ID         string      `xml:"id,attr"`
	Name       string      `xml:"name,attr"`
	Failures   int         `xml:"failures,attr"`
	Time       float64     `xml:"time,attr"`
	Tests      int         `xml:"tests,attr"`
	Testsuites []testsuite `xml:"testsuite"`
}

type testsuite struct {
	ID        string     `xml:"id,attr"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Time      float64    `xml:"time,attr"`
	Testcases []testcase `xml:"testcase"`
	Failure   *failure   `xml:"failure,omitempty"`
}

type testcase struct {
	ID      string   `xml:"id,attr"`
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

// ParseJSONResult Print the result to the console
func ParseJSONResult(baseResult *Element) []byte {
	jsonResult, _ := json.MarshalIndent(baseResult, "", "  ")

	return jsonResult
}

// ParseJUnitResult Print the result to the console
func ParseJUnitResult(baseResult *Element) []byte {

	testName := time.Now().Format("2006-01-02 15:04")
	result := XMLRoot{
		Name:     testName,
		ID:       testName,
		Failures: baseResult.Failures,
		Tests:    baseResult.TestCount,
		Time:     baseResult.ExecutionTime.Seconds(),
	}

	for k, v := range baseResult.SubTests {
		newTestSuite := testsuite{
			ID:       strconv.Itoa(k),
			Time:     v.ExecutionTime.Seconds(),
			Failures: v.Failures,
			Tests:    v.TestCount,
			Name:     strings.Replace(v.Name, ".", ":", -1),
		}

		if v.Failure != "" {
			newTestSuite.Failure = &failure{
				Type:    "ERROR",
				Message: v.Failure,
			}
		}

		flattenSubTests := v.SubTests.Flat()

		padding := iterativeDigitsCount(len(flattenSubTests))
		for ik, iv := range flattenSubTests {
			newTestCase := testcase{
				ID:   strconv.Itoa(ik),
				Name: fmt.Sprintf("[%0"+strconv.Itoa(padding)+"d] %s", ik, strings.Replace(iv.Name, ".", ":", -1)),
			}

			// only save the time if a test has no sub tests, so the total times are only included once in the report
			if len(iv.SubTests) == 0 {
				newTestCase.Time = iv.ExecutionTime.Seconds()
			} else {
				newTestCase.Time = 0
			}

			if iv.Failures > 0 {
				newTestCase.Failure = &failure{Type: "ERROR"}
				for _, jv := range iv.GetLog() {
					newTestCase.Failure.Message = fmt.Sprintf("%s\n\n%s", newTestCase.Failure.Message, jv)
				}
				if len(newTestCase.Failure.Message) == 0 {
					newTestCase.Failure = nil
				}
			}

			newTestSuite.Testcases = append(newTestSuite.Testcases, newTestCase)
		}
		result.Testsuites = append(result.Testsuites, newTestSuite)
	}

	xmlResult, _ := xml.MarshalIndent(result, "", "  ")

	return xmlResult
}

func iterativeDigitsCount(number int) int {
	count := 0
	for number != 0 {
		number /= 10
		count++
	}
	return count
}
