package report

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/programmfabrik/apitest/pkg/lib/cjson"
	"github.com/programmfabrik/apitest/pkg/lib/compare"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func TestReportStructure(t *testing.T) {
	r := NewReport()
	r.Root().NoLogTime = true

	r.Root().NewChild("Level 1 - 1").Leave(true)

	child := r.Root().NewChild("Level 1 - 2")

	child.NewChild("Level 2 - 1").Leave(true)

	child2 := child.NewChild("Level 2 - 2")
	child2.NewChild("Level 3 - 1").Leave(true)
	child2.Leave(true)
	child.Leave(true)

	if r.root.SubTests[1].SubTests[0].Name != "Level 2 - 1" {
		t.Error(r.root.SubTests[1].SubTests[0].Name, " != Level 2 - 1")
	}
	if r.root.SubTests[1].SubTests[1].SubTests[0].Name != "Level 3 - 1" {
		t.Error(r.root.SubTests[1].SubTests[0].Name, " != Level 3 - 1")
	}
}

func TestReportGetJSONResult(t *testing.T) {
	r := NewReport()
	r.Root().NoLogTime = true
	r.Root().NewChild("Level 1 - 1").Leave(false)

	child := r.Root().NewChild("Level 1 - 2")

	child.NewChild("Level 2 - 1").Leave(true)

	child2 := child.NewChild("Level 2 - 2")
	child3 := child2.NewChild("Level 3 - 1")
	time.Sleep(50 * time.Microsecond)
	child3.Leave(false)
	child2.Leave(true)
	child.Leave(true)

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
	r.Root().NoLogTime = true
	r.Root().NewChild("Level 1 - 1").Leave(false)

	child := r.Root().NewChild("Level 1 - 2")
	child.NewChild("Level 2 - 1").Leave(true)

	child2 := child.NewChild("Level 2 - 2")
	child2.NewChild("Level 3 - 1").Leave(false)
	time.Sleep(50 * time.Microsecond)
	child2.Leave(true)
	child.Leave(true)

	jsonResult := r.GetTestResult(ParseJUnitResult)
	expResult := `<testsuites failures="2" tests="3">
	<testsuite id="0" name="Level 1 - 1" tests="1" failures="1" ></testsuite>
	<testsuite id="1" name="Level 1 - 2" tests="2" failures="1">
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
	r.Root().NoLogTime = true

	child := r.Root().NewChild("Level 1 - 1")
	child.SaveToReportLog("Log Level 1 - 1")
	child.Leave(true)

	child2 := r.Root().NewChild("Level 1 - 2")

	child21 := child2.NewChild("Level 2 - 1")
	child21.SaveToReportLog("Log Level 2 - 1 [1]")
	child21.SaveToReportLog("Log Level 2 - 1 [2]")
	child21.SaveToReportLog("Log Level 2 - 1 [3]")
	child21.Leave(true)
	child22 := child2.NewChild("Level 2 - 2")
	child22.SaveToReportLog("Log Level 2 - 2 [1]")
	child3 := child22.NewChild("Level 3 - 1")
	child3.SaveToReportLog("Log Level 3 - 1 [1]")
	child3.Leave(true)
	child22.Leave(true)
	child2.Leave(true)

	expectedReport := []string{"Log Level 1 - 1", "Log Level 2 - 1 [1]", "Log Level 2 - 1 [2]", "Log Level 2 - 1 [3]", "Log Level 2 - 2 [1]", "Log Level 3 - 1 [1]"}
	realReport := r.GetLog()

	for k, v := range expectedReport {
		if realReport[k] != v {
			t.Fatalf("Reports did not match.\n Expected:\n%s\nGot:\n%s", expectedReport, realReport)
		}
	}
}
