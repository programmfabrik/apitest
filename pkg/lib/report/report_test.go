package report

import (
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"

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

	var expJ, realJ interface{}

	util.Unmarshal(jsonResult, &realJ)
	util.Unmarshal(expResult, &expJ)

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
	child3 := child2.NewChild("Level 3 - 1")
	child3.Leave(false)
	time.Sleep(50 * time.Microsecond)
	child2.Leave(true)
	child.Leave(true)

	jsonResult := r.GetTestResult(ParseJUnitResult)
	expResult := `<testsuites failures="2" tests="3">
	<testsuite id="0" name="Level 1 - 1" tests="1" failures="1" ></testsuite>
	<testsuite id="1" name="Level 1 - 2" tests="2" failures="1">
		<testcase id="0" name="[0] Level 2 - 1"></testcase>
		<testcase id="1" name="[1] Level 2 - 2"></testcase>
		<testcase id="2" name="[2] Level 3 - 1"></testcase>
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

	expJBytes, _ := json.Marshal(expX)
	realJBytes, _ := json.Marshal(realX)

	var expJ, realJ interface{}

	util.Unmarshal(expJBytes, &expJ)
	util.Unmarshal(realJBytes, &realJ)

	equal, _ := compare.JsonEqual(expJ, realJ, compare.ComparisonContext{})

	if !equal.Equal {
		//		t.Error(equal.Failures)
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

func TestReportGetStatsResult(t *testing.T) {
	r := NewReport()
	r.StatsGroups = 2
	r.Version = "dummy"

	child := r.Root().NewChild("manifest1.json")
	time.Sleep(10 * time.Millisecond)
	subchild := child.NewChild("manifest1_sub1")
	time.Sleep(20 * time.Millisecond)
	subchild.Leave(true)
	subchild2 := child.NewChild("manifest1_sub2")
	time.Sleep(30 * time.Millisecond)
	subchild2.Leave(true)
	child.Leave(true)

	child2 := r.Root().NewChild("manifest2.json")
	time.Sleep(50 * time.Millisecond)
	subchild3 := child2.NewChild("manifest2_sub1")
	time.Sleep(150 * time.Millisecond)
	subchild3.Leave(true)
	subchild4 := child2.NewChild("manifest2_sub2")
	time.Sleep(200 * time.Millisecond)
	subchild4.Leave(true)
	child2.Leave(true)

	child3 := r.Root().NewChild("manifest3.json")
	time.Sleep(50 * time.Millisecond)
	subchild5 := child3.NewChild("manifest3_sub1")
	time.Sleep(50 * time.Millisecond)
	subchild5.Leave(true)
	subchild6 := child3.NewChild("manifest3_sub2")
	time.Sleep(50 * time.Millisecond)
	subchild6.Leave(true)
	child3.Leave(true)

	jsonResult := r.GetTestResult(ParseJSONStatsResult)
	var statsRep statsReport
	util.Unmarshal(jsonResult, &statsRep)

	if statsRep.Version != r.Version {
		t.Fatalf("Got version %s, expected %s", statsRep.Version, r.Version)
	}
	if statsRep.Groups != r.StatsGroups {
		t.Fatalf("Got %d groups, expected %d", statsRep.Groups, r.StatsGroups)
	}
	if len(statsRep.Manifests) != 3 {
		t.Fatalf("Got %d manifests, expected 3", len(statsRep.Manifests))
	}
	if statsRep.Manifests[0].Group != 1 {
		t.Fatalf("Manifest 1 in group %d, expected to be in 1", statsRep.Manifests[0].Group)
	}
	if statsRep.Manifests[1].Group != 0 {
		t.Fatalf("Manifest 2 in group %d, expected to be in 0", statsRep.Manifests[1].Group)
	}
	if statsRep.Manifests[2].Group != 1 {
		t.Fatalf("Manifest 3 in group %d, expected to be in 1", statsRep.Manifests[2].Group)
	}
}
