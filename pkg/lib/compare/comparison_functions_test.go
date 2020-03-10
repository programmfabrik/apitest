package compare

import (
	"fmt"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/util"
	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func TestComparison(t *testing.T) {
	testData := []struct {
		name      string
		left      util.JSONObject
		right     util.JSONObject
		eEqual    bool
		eFailures []Failure
	}{
		{
			name: "Should be equal",
			left: util.JSONObject{
				"array": util.JSONArray{
					"val2",
					"val3",
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					"val2",
					"val3",
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any string",
			left: util.JSONObject{
				"stringerino:control": util.JSONObject{
					"must_exist": true,
					"is_string":  true,
				},
			},

			right: util.JSONObject{
				"stringerino": "not equal",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String matches regex",
			left: util.JSONObject{
				"stringerino:control": util.JSONObject{
					"match":     "\\d+\\..+",
					"is_string": true,
				},
			},
			right: util.JSONObject{
				"stringerino": "123.abc",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String does not match regex",
			left: util.JSONObject{
				"stringerino:control": util.JSONObject{
					"match":     "\\d+\\.\\d+",
					"is_string": true,
				},
			},
			right: util.JSONObject{
				"stringerino": "xyz-456",
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "stringerino",
					Message: "does not match regex '\\d+\\.\\d+'",
				},
			},
		},
		{
			name: "String match with invalid regex (must fail)",
			left: util.JSONObject{
				"stringerino:control": util.JSONObject{
					"match":     ".+[",
					"is_string": true,
				},
			},
			right: util.JSONObject{
				"stringerino": "",
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "stringerino",
					Message: "could not match regex '.+[': 'error parsing regexp: missing closing ]: `[`'",
				},
			},
		},
		{
			name: "String match tried on integer (must fail)",
			left: util.JSONObject{
				"numberino:control": util.JSONObject{
					"match": ".+",
				},
			},
			right: util.JSONObject{
				"numberino": util.JSONNumber(123),
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "numberino",
					Message: "should be 'String' for regex match but is 'Number'",
				},
			},
		},
		{
			name: "There should be any number",
			left: util.JSONObject{
				"numberino:control": util.JSONObject{
					"must_exist": true,
					"is_number":  true,
				},
			},
			right: util.JSONObject{
				"numberino": util.JSONNumber(99999999),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any bool",
			left: util.JSONObject{
				"boolerino:control": util.JSONObject{
					"must_exist": true,
					"is_bool":    true,
				},
			},
			right: util.JSONObject{
				"boolerino": util.JSONBool(false),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any array",
			left: util.JSONObject{
				"arrayerino:control": util.JSONObject{
					"must_exist": true,
					"is_array":   true,
				},
			},
			right: util.JSONObject{
				"arrayerino": util.JSONArray(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be an empty object",
			left: util.JSONObject{
				"objecterino": util.JSONObject{},
				"objecterino:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"objecterino": util.JSONObject{"1": 1, "2": 2, "3": 3},
			},
			eEqual: false,
			eFailures: []Failure{
				{"objecterino", `extra elements found in object`},
			},
		},
		{
			name: "There should be empty array",
			left: util.JSONObject{
				"arrayerino": util.JSONArray{},
				"arrayerino:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"arrayerino": util.JSONArray{"1", "2", "3"},
			},
			eEqual: false,
			eFailures: []Failure{
				{"arrayerino", `extra elements found in array`},
			},
		},
		{
			name: "There should be any object",
			left: util.JSONObject{
				"objecterino:control": util.JSONObject{
					"must_exist": true,
					"is_object":  true,
				},
			},
			right: util.JSONObject{
				"objecterino": util.JSONObject(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Token match with wrong order",
			left: util.JSONObject{
				"tokens": util.JSONArray{
					util.JSONObject{
						"suggest": "<b>a</b>",
					},
					util.JSONObject{
						"suggest": "<b>a</b>b",
					},
					util.JSONObject{
						"suggest": "<b>a</b>bc",
					},
				},
			},
			right: util.JSONObject{
				"tokens": util.JSONArray{
					util.JSONObject{
						"suggest": "<b>a</b>",
					},
					util.JSONObject{
						"suggest": "<b>a</b>bc",
					},
					util.JSONObject{
						"suggest": "<b>a</b>b",
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be no object",
			left: util.JSONObject{
				"objecterino:control": util.JSONObject{
					"must_not_exist": true,
				},
			},
			right:     util.JSONObject{},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be no object but it exists",
			left: util.JSONObject{
				"objecterino:control": util.JSONObject{
					"must_not_exist": true,
				},
			},
			right: util.JSONObject{
				"objecterino": util.JSONObject(nil),
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be no deeper object but it exists",
			left: util.JSONObject{
				"it": util.JSONArray{
					util.JSONObject{
						"objecterino:control": util.JSONObject{
							"must_not_exist": true,
						},
					},
				},
			},
			right: util.JSONObject{
				"it": util.JSONArray{
					util.JSONObject{
						"objecterino": util.JSONString("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "it[0].objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be no deeper object but it exists2",
			left: util.JSONObject{
				"it": util.JSONArray{
					util.JSONObject{
						"objecterino:control": util.JSONObject{
							"must_not_exist": true,
						},
					},
				},
				"it:control": util.JSONObject{
					"order_matters": true,
				},
			},
			right: util.JSONObject{
				"it": util.JSONArray{
					util.JSONObject{
						"objecterino": util.JSONString("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "it[0].objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be a exact object match",
			left: util.JSONObject{
				"objecterino": util.JSONObject{
					"1": util.JSONNumber(1),
					"2": util.JSONNumber(2),
					"3": util.JSONNumber(3),
				},
				"objecterino:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"objecterino": util.JSONObject{
					"1": util.JSONNumber(1),
					"3": util.JSONNumber(3),
					"2": util.JSONNumber(2),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be a exact object match even if order is mixed",
			left: util.JSONObject{
				"objecterino": util.JSONObject{
					"1": util.JSONNumber(1),
					"2": util.JSONNumber(2),
					"3": util.JSONNumber(3),
				},
				"objecterino:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"objecterino": util.JSONObject{
					"2": util.JSONNumber(2),
					"3": util.JSONNumber(3),
					"1": util.JSONNumber(1),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Exact match is not present",
			left: util.JSONObject{
				"MYobjecterino": util.JSONObject{
					"1": util.JSONNumber(1),
					"2": util.JSONNumber(2),
					"3": util.JSONNumber(3),
				},
				"MYobjecterino:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"MYobjecterino": util.JSONObject{
					"2": util.JSONNumber(2),
					"4": util.JSONNumber(4),
					"1": util.JSONNumber(1),
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "MYobjecterino.3",
					Message: "was not found, but should exist",
				},
				{
					Key:     "MYobjecterino",
					Message: "extra elements found in object",
				},
			},
		},
		{
			name: "Not all contained",
			left: util.JSONObject{
				"array": util.JSONArray{
					"val2",
					"val3",
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					"val1",
					"val2",
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "array[1]",
					Message: "Got 'val1', expected 'val3'",
				},
			},
		},
		{
			name: "Wrong order",
			left: util.JSONObject{
				"array": util.JSONArray{
					"val3",
					"val2",
				},
				"array:control": util.JSONObject{
					"order_matters": true,
					"no_extra":      true,
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					"val2",
					"val3",
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "array[0]",
					Message: "Got 'val2', expected 'val3'",
				},
				{
					Key:     "array[1]",
					Message: "Got 'val3', expected 'val2'",
				},
			},
		},
		{
			name: "Wrong order deeper with map",
			left: util.JSONObject{
				"array": util.JSONObject{
					"inner": util.JSONObject{
						"deeper": util.JSONArray{
							"val4",
							"val5",
						},
						"deeper:control": util.JSONObject{
							"order_matters": true,
						},
					},
				},
			},
			right: util.JSONObject{
				"array": util.JSONObject{
					"inner": util.JSONObject{
						"deeper": util.JSONArray{
							"val5",
							"val4",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "array.inner.deeper[0]",
					Message: "Got 'val5', expected 'val4'",
				},
				{
					Key:     "array.inner.deeper[1]",
					Message: "Got 'val4', expected 'val5'",
				},
			},
		},
		{
			name: "Right error message for array",
			left: util.JSONObject{
				"body": util.JSONArray{
					util.JSONObject{
						"henk": "denk",
					},
				},
			},
			right: util.JSONObject{
				"body": util.JSONArray{
					util.JSONObject{},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "body[0].henk",
					Message: "was not found, but should exist",
				},
			},
		},
		{
			name: "All fine deeper with arrays",
			left: util.JSONObject{
				"array": util.JSONArray{
					util.JSONArray{
						util.JSONArray{
							"val9",
							"val10",
						},
					},
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					util.JSONArray{
						util.JSONArray{
							"val9",
							"val10",
						},
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check array length",
			left: util.JSONObject{
				"array:control": util.JSONObject{
					"element_count": 3,
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					util.JSONArray{
						"val9",
						"val10",
					},
					util.JSONArray{
						"val9",
						"val10",
					},
					util.JSONArray{
						"val9",
						"val10",
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check array length and fail",
			left: util.JSONObject{
				"array:control": util.JSONObject{
					"element_count": 2,
				},
				"array": util.JSONArray{
					util.JSONArray{},
				},
			},
			right: util.JSONObject{
				"array": util.JSONArray{
					util.JSONArray{
						util.JSONArray{
							"val9",
							"val10",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "array",
					Message: "length of the actual response array '1' != '2' expected length",
				},
			},
		},
		{
			name: "Check body no extra",
			left: util.JSONObject{
				"body": util.JSONArray{
					util.JSONObject{
						"pool": util.JSONObject{
							"reference": "system:root",
						},
					},
				},
				"body:control": util.JSONObject{
					"no_extra": true,
				},
			},
			right: util.JSONObject{
				"body": util.JSONArray{
					util.JSONObject{
						"pool": util.JSONObject{
							"reference":  "system:root",
							"reference2": "system:root",
						},
					},
					util.JSONObject{
						"pool": util.JSONObject{
							"reference": "system:root",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []Failure{
				{
					Key:     "body",
					Message: "extra elements found in array",
				},
			},
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			equal, err := JSONEqual(data.left, data.right, ComparisonContext{})
			if err != nil {
				t.Fatal(err)
			}

			if equal.Equal != data.eEqual {
				t.Errorf("Equal: got '%t', expected '%t'", equal.Equal, data.eEqual)
			}

			if (equal.Failures != nil && data.eFailures == nil) || (equal.Failures == nil && data.eFailures != nil) {
				t.Errorf("Failure: got '%v', expected '%v'", equal.Failures, data.eFailures)
				return
			}

			wantFailures := []string{}
			for _, v := range data.eFailures {
				wantFailures = append(wantFailures, fmt.Sprintf("%s", v))
			}
			haveFailures := []string{}
			for _, v := range equal.Failures {
				haveFailures = append(haveFailures, fmt.Sprintf("%s", v))
			}

			go_test_utils.AssertStringArraysEqualNoOrder(t, wantFailures, haveFailures)

			if t.Failed() {
				t.Log("EXPECTED ", wantFailures)
				t.Log("GOT      ", haveFailures)
			}
		})
	}
}
