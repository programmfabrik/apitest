package report

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"github.com/programmfabrik/fylr-apitest/lib/compare"
	"github.com/programmfabrik/fylr-apitest/lib/util"
)

func TestReportStructure(t *testing.T) {
	r := NewReport()

	r.NewChild("Level 1 - 1")
	r.LeaveChild(true)
	r.NewChild("Level 1 - 2")
	r.NewChild("Level 2 - 1")
	r.LeaveChild(true)
	r.NewChild("Level 2 - 2")
	r.NewChild("Level 3 - 1")
	r.LeaveChild(true)
	r.LeaveChild(true)
	r.LeaveChild(true)
	r.LeaveChild(true)

	if r.root.SubTests[1].SubTests[0].Name != "Level 2 - 1" {
		t.Error(r.root.SubTests[1].SubTests[0].Name, " != Level 2 - 1")
	}
	if r.root.SubTests[1].SubTests[1].SubTests[0].Name != "Level 3 - 1" {
		t.Error(r.root.SubTests[1].SubTests[0].Name, " != Level 3 - 1")
	}
}

func TestReportGetJSONResult(t *testing.T) {
	r := NewReport()
	r.NewChild("Level 1 - 1")
	r.LeaveChild(false)
	r.NewChild("Level 1 - 2")
	r.NewChild("Level 2 - 1")
	r.LeaveChild(true)
	r.NewChild("Level 2 - 2")
	r.NewChild("Level 3 - 1")
	time.Sleep(50 * time.Microsecond)
	r.LeaveChild(false)
	r.LeaveChild(true)
	r.LeaveChild(true)
	r.LeaveChild(true)

	jsonResult := r.GetTestResult(ParseJSONResult)
	expResult := []byte(`{
 "failures": 2,
 "sub_tests": [
  {
   "failures": 1,
   "name": "Level 1 - 1"
  },
  {
   "failures": 1,
   "name": "Level 1 - 2",
   "sub_tests": [
    {
     "failures": 0,
     "name": "Level 2 - 1"
    },
    {
     "failures": 1,
     "name": "Level 2 - 2",
     "sub_tests": [
      {
       "failures": 1,
       "name": "Level 3 - 1"
      }
     ]
    }
   ]
  }
 ]
}`)

	var expJ, realJ util.GenericJson

	cjson.Unmarshal(jsonResult, &realJ)
	cjson.Unmarshal(expResult, &expJ)

	equal, _ := compare.JsonEqual(expJ, realJ, compare.ComparisonContext{})

	if !equal.Equal {
		t.Errorf("Wanted:\n%s\n\nGot:\n%s", expResult, jsonResult)
		t.Fail()
	}
}
func TestReportGetJUnitResult(t *testing.T) {
	r := NewReport()
	r.NewChild("Level 1 - 1")
	r.LeaveChild(false)
	r.NewChild("Level 1 - 2")
	r.NewChild("Level 2 - 1")
	r.LeaveChild(true)
	r.NewChild("Level 2 - 2")
	r.NewChild("Level 3 - 1")
	time.Sleep(50 * time.Microsecond)
	r.LeaveChild(false)
	r.LeaveChild(true)
	r.LeaveChild(true)

	jsonResult := r.GetTestResult(ParseJUnitResult)
	expResult := `<testsuites failures="2" tests="0">
	<testsuite id="0" name="Level 1 - 1" tests="0" failures="1" ></testsuite>
	<testsuite id="1" name="Level 1 - 2" tests="0" failures="1">
		<testcase id="0" name="Level 2 - 1" ></testcase>
		<testcase id="1" name="Level 2 - 2" >
			<failure message="" type="ERROR"></failure>
		</testcase>
	</testsuite>
</testsuites>`

	var expX, realX XMLRoot

	xml.Unmarshal([]byte(expResult), &expX)
	xml.Unmarshal(jsonResult, &realX)

	realX.Id = ""
	realX.Name = ""
	realX.Time = 0

	for k, v := range realX.Testsuites {
		realX.Testsuites[k].Time = 0
		for ik, _ := range v.Testcases {
			realX.Testsuites[k].Testcases[ik].Time = 0
		}
	}

	expJBytes, _ := cjson.Marshal(expX)
	realJBytes, _ := cjson.Marshal(realX)

	var expJ, realJ util.GenericJson

	cjson.Unmarshal(expJBytes, &expJ)
	cjson.Unmarshal(realJBytes, &realJ)

	equal, _ := compare.JsonEqual(expJ, realJ, compare.ComparisonContext{})

	if !equal.Equal {
		t.Error(equal.Failures)
		t.Errorf("Wanted:\n%s\n\nGot:\n%s", expJBytes, realJBytes)
		t.Fail()
	}
}

func TestReportLog(t *testing.T) {
	r := NewReport()

	r.NewChild("Level 1 - 1")
	r.SaveToReportLog("Log Level 1 - 1")
	r.LeaveChild(true)

	r.NewChild("Level 1 - 2")
	r.NewChild("Level 2 - 1")
	r.SaveToReportLog("Log Level 2 - 1 [1]")
	r.SaveToReportLog("Log Level 2 - 1 [2]")
	r.SaveToReportLog("Log Level 2 - 1 [3]")
	r.LeaveChild(true)
	r.NewChild("Level 2 - 2")
	r.SaveToReportLog("Log Level 2 - 2 [1]")
	r.NewChild("Level 3 - 1")
	r.SaveToReportLog("Log Level 3 - 1 [1]")
	r.LeaveChild(true)
	r.LeaveChild(true)
	r.LeaveChild(true)
	r.LeaveChild(true)

	expectedReport := []string{"Log Level 1 - 1", "Log Level 2 - 1 [1]", "Log Level 2 - 1 [2]", "Log Level 2 - 1 [3]", "Log Level 2 - 2 [1]", "Log Level 3 - 1 [1]"}
	realReport := r.GetLog()

	for k, v := range expectedReport {
		if realReport[k] != v {
			t.Fatalf("Reports did not match.\n Expected:\n%s\nGot:\n%s", expectedReport, realReport)
		}
	}
}
