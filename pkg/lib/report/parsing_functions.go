package report

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/programmfabrik/golib"
)

type xmlRoot struct {
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
	Failure   *failure   `xml:"failure,omitempty"`
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

type statsReport struct {
	StartedAt time.Time            `json:"started_at"`
	EndedAt   time.Time            `json:"ended_at"`
	Version   string               `json:"version"`
	Groups    statsGroups          `json:"groups"`
	Manifests []statsReportElement `json:"manifests"`
	User      string               `json:"user"`
	Path      string               `json:"path"`
}

type statsReportElement struct {
	Group     int       `json:"group"`
	Path      string    `json:"path"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	RuntimeMS int64     `json:"runtime_ms"`
}

type statsGroup struct {
	Number     int           `json:"number"`
	Runtime    time.Duration `json:"-"`
	RuntimeStr string        `json:"runtime"`
}
type statsGroups []statsGroup

func (groups statsGroups) getLowestRuntimeGroup() (group int) {
	if len(groups) < 1 {
		panic("No groups to check lowest duration")
	}
	lowestDuration := 9999 * time.Hour
	for i, g := range groups {
		if g.Runtime < lowestDuration {
			group = i
			lowestDuration = g.Runtime
		}
	}
	return
}

// ParseJSONResult Print the result to the console
func ParseJSONResult(baseResult *ReportElement) []byte {
	jsonResult, _ := golib.JsonBytesIndent(baseResult, "", "  ")

	return jsonResult
}

// ParseJSONResult Print the result to the console
func parseJSONStatsResult(baseResult *ReportElement) []byte {

	currUsername := "unknown"
	currUser, _ := user.Current()
	if currUser != nil {
		currUsername = currUser.Username
	}
	currPath, _ := os.Getwd()

	stats := statsReport{
		StartedAt: baseResult.StartTime,
		EndedAt:   baseResult.StartTime.Add(baseResult.ExecutionTime),
		Version:   baseResult.report.Version,
		Manifests: []statsReportElement{},
		User:      currUsername,
		Path:      currPath,
	}

	stats.Groups = make([]statsGroup, baseResult.report.StatsGroups)
	for i := range baseResult.report.StatsGroups {
		stats.Groups[i] = statsGroup{
			Number:  i,
			Runtime: 0,
		}
	}

	sort.SliceStable(baseResult.SubTests, func(i, j int) bool {
		return baseResult.SubTests[i].ExecutionTime > baseResult.SubTests[j].ExecutionTime
	})

	for _, r := range baseResult.SubTests {
		currGroup := stats.Groups.getLowestRuntimeGroup()
		stats.Manifests = append(stats.Manifests, statsReportElement{
			Group:     currGroup,
			StartedAt: r.StartTime,
			EndedAt:   r.StartTime.Add(r.ExecutionTime),
			RuntimeMS: r.ExecutionTime.Milliseconds(),
			Path:      r.Name,
		})
		stats.Groups[currGroup].Runtime += r.ExecutionTime
	}

	sort.SliceStable(stats.Manifests, func(i, j int) bool {
		return stats.Manifests[i].StartedAt.Before(stats.Manifests[j].StartedAt)
	})

	for i, g := range stats.Groups {
		stats.Groups[i].RuntimeStr = g.Runtime.String()
	}

	jsonResult, _ := golib.JsonBytesIndent(stats, "", "  ")

	return jsonResult
}

// parseJUnitResult Print the result to the console
func parseJUnitResult(baseResult *ReportElement) []byte {

	testName := time.Now().Format("2006-01-02 15:04")
	result := xmlRoot{
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
				Id:   strconv.Itoa(ik),
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
				for _, jv := range iv.getLog() {
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
