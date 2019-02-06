package cjson

import (
	"fmt"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/compare"
	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
	"github.com/programmfabrik/fylr-apitest/lib/util"
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
			eError:  fmt.Errorf("Couldn't find offset %d within the input.", -6),
		},
		{
			iOffset: 999,
			eError:  fmt.Errorf("Couldn't find offset %d within the input.", 999),
		},
	}

	for _, v := range testCases {
		oLine, oCharacter, oErr := lineAndCharacter(v.iString, v.iOffset)

		test_utils.AssertErrorEquals(t, oErr, v.eError)
		if oErr == nil {
			test_utils.AssertIntEquals(t, oLine, v.eLine)
			test_utils.AssertIntEquals(t, oCharacter, v.eCharacter)
		}
	}

}

func TestRealWorldJsonError(t *testing.T) {

	testCases := []struct {
		iJson  string
		eError error
	}{
		{
			`{
"hallo":2,
}`,
			fmt.Errorf(`Cannot parse JSON '
1: {
2: "hallo":2,
3: }
' schema due to a syntax error at line 3, character 1: invalid character '}' looking for beginning of object key string`),
		},
		{
			`{
"hallo":2,
welt:1
}`,
			fmt.Errorf(`Cannot parse JSON '
1: {
2: "hallo":2,
3: welt:1
4: }
' schema due to a syntax error at line 3, character 1: invalid character 'w' looking for beginning of object key string`),
		},
		{
			`{
"hallo": 2,
"welt": "string
}`,
			fmt.Errorf(`Cannot parse JSON '
1: {
2: "hallo": 2,
3: "welt": "string
4: }
' schema due to a syntax error at line 4, character 0: invalid character '\n' in string literal`),
		},
	}

	for _, v := range testCases {
		var out util.JsonObject
		oErr := Unmarshal([]byte(v.iJson), &out)

		test_utils.AssertErrorEquals(t, oErr, v.eError)
	}

}

func TestRemoveComments(t *testing.T) {
	testCases := []struct {
		iJson string
		eOut  util.JsonObject
	}{
		{
			`{
"hallo":2
}`,
			util.JsonObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo":2
}`,
			util.JsonObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo":2
## line 2

#line2
}`,
			util.JsonObject{
				"hallo": float64(2),
			},
		},
		{
			`{
"hallo":2,
## line 2

#line2
"hey":"ha"
}`,
			util.JsonObject{
				"hallo": float64(2),
				"hey":   "ha",
			},
		},
	}

	for _, v := range testCases {
		var out util.JsonObject
		Unmarshal([]byte(v.iJson), &out)
		for k, v := range v.eOut {
			if out[k] != v {
				t.Errorf("[%s] Have '%f' != '%f' want", k, out[k], v)
			}
		}
	}

}

func TestCJSONUnmarshalSyntaxErr(t *testing.T) {
	testCases := []struct {
		cjsonString string
		eObject     util.JsonObject
		eError      error
	}{
		{
			`{"hallo":3}`,
			util.JsonObject{
				"hallo": float64(3),
			},
			nil,
		},
		{
			cjsonString: `{"hallo":3
"fail":"s"}`,
			eError: fmt.Errorf(`Cannot parse JSON '
1: {"hallo":3
2: "fail":"s"}
' schema due to a syntax error at line 2, character 1: invalid character '"' after object key:value pair`),
		},
		{
			cjsonString: `{"hallo":3,
"fail":"s",
"simon":{
	"hey":"e
}
}`,
			eError: fmt.Errorf(`Cannot parse JSON '
1: {"hallo":3,
2: "fail":"s",
3: "simon":{
4: 	"hey":"e
5: }
6: }
' schema due to a syntax error at line 5, character 0: invalid character '\n' in string literal`),
		},
		{
			cjsonString: `{"hallo":3,
"fail":"s",


"simon":{
	"hey":"e
}
}`,
			eError: fmt.Errorf(`Cannot parse JSON '
1: {"hallo":3,
2: "fail":"s",
5: "simon":{
6: 	"hey":"e
7: }
8: }
' schema due to a syntax error at line 7, character 0: invalid character '\n' in string literal`),
		},
		{
			cjsonString: `{"hallo":3,
"fail":"s",
"simon":{
	"hey":"e
}







}`,
			eError: fmt.Errorf(`Cannot parse JSON '
 1: {"hallo":3,
 2: "fail":"s",
 3: "simon":{
 4: 	"hey":"e
 5: }
13: }
' schema due to a syntax error at line 5, character 0: invalid character '\n' in string literal`),
		},
	}

	for _, v := range testCases {
		oObject := util.JsonObject{}
		oErr := Unmarshal([]byte(v.cjsonString), &oObject)

		test_utils.AssertErrorEquals(t, oErr, v.eError)
		if oErr == nil {

			equal, _ := compare.JsonEqual(v.eObject, oObject, compare.ComparisonContext{})

			if !equal.Equal {
				t.Errorf("Objects are not the same.\nExpected: %v\nGot: %v", v.eObject, oObject)
			}
		}
	}
}

func TestCJSONUnmarshalTypeErr(t *testing.T) {
	cjsonString := `{"Name":3}`
	cjsonStringLines := `
1: {"Name":3}
`

	type expectedStructure struct {
		Name string `json:"name"`
	}

	var oObject expectedStructure

	oErr := Unmarshal(
		[]byte(cjsonString),
		&oObject,
	)

	test_utils.AsserErrorEqualsAny(
		t,
		oErr,
		[]error{
			fmt.Errorf("In JSON '%s', the type 'number' cannot be converted into the Go 'string' type on struct '', field ''. See input file line 1, character 9", cjsonStringLines),
			fmt.Errorf("In JSON '%s', the type 'number' cannot be converted into the Go 'string' type on struct 'expectedStructure', field 'name'. See input file line 1, character 9", cjsonStringLines),
		},
	)
}
