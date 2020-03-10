package cjson

import (
	"fmt"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/util"
	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func init() {
	coloredError = false
}

func TestLineAndCharacter(t *testing.T) {
	testCases := []struct {
		iString    string
		iOffset    int
		eLine      int
		eCharacter int
		eError     error
	}{
		{
			"ab\ncdefg",
			5,
			2,
			2,
			nil,
		},
		{
			"ab\nc\nde\nfg",
			5,
			3,
			0,
			nil,
		},
		{
			"a\n\nb\ncdefg",
			6,
			4,
			1,
			nil,
		},
		{
			iOffset: -6,
			eError:  fmt.Errorf("couldn't find offset %d within the input", -6),
		},
		{
			iOffset: 999,
			eError:  fmt.Errorf("couldn't find offset %d within the input", 999),
		},
	}

	for _, v := range testCases {
		oLine, oCharacter, oErr := lineAndCharacter(v.iString, v.iOffset)

		go_test_utils.AssertErrorEquals(t, oErr, v.eError)
		if oErr == nil {
			go_test_utils.AssertIntEquals(t, oLine, v.eLine)
			go_test_utils.AssertIntEquals(t, oCharacter, v.eCharacter)
		}
	}

}

func TestRemoveComments(t *testing.T) {
	testCases := []struct {
		iJSON string
		eOut  util.JSONObject
	}{
		{
			`{
"hallo": 2
}`,
			util.JSONObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo": 2
}`,
			util.JSONObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo": 2
## line 2

#line2
}`,
			util.JSONObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo": 2,
## line 2

#line2
"hey": "ha"
}`,
			util.JSONObject{
				"hallo": float64(2),
				"hey":   "ha",
			},
		},
	}

	for _, v := range testCases {
		var out util.JSONObject
		Unmarshal([]byte(v.iJSON), &out)
		for k, v := range v.eOut {
			if out[k] != v {
				t.Errorf("[%s] Got '%f', expected '%f'", k, out[k], v)
			}
		}
	}

}

func TestCJSONUnmarshalTypeErr(t *testing.T) {
	cjsonString := `{"Name": 3}`
	cjsonStringLines := `
1: {"Name": 3}
`

	type expectedStructure struct {
		Name string `json: "name"`
	}

	var oObject expectedStructure

	oErr := Unmarshal(
		[]byte(cjsonString),
		&oObject,
	)

	go_test_utils.AsserErrorEqualsAny(
		t,
		oErr,
		[]error{
			fmt.Errorf("In JSON '%s', the type 'number' cannot be converted into the Go 'string' type on struct '', field ''. See input file line 1, character 10", cjsonStringLines),
			fmt.Errorf("In JSON '%s', the type 'number' cannot be converted into the Go 'string' type on struct 'expectedStructure', field 'name'. See input file line 1, character 10", cjsonStringLines),
			fmt.Errorf("In JSON '%s', the type 'number' cannot be converted into the Go 'string' type on struct 'expectedStructure', field 'Name'. See input file line 1, character 10", cjsonStringLines),
		},
	)
}
