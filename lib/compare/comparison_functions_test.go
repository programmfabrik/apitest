package compare

import (
	"fmt"
	"testing"

	"github.com/programmfabrik/go-test-utils"

	"github.com/programmfabrik/fylr-apitest/lib/util"
)

func TestComparison(t *testing.T) {
	testData := []struct {
		name      string
		left      util.JsonObject
		right     util.JsonObject
		eEqual    bool
		eFailures []CompareFailure
	}{
		{
			name: "Should be equal",
			left: util.JsonObject{
				"array": util.JsonArray{
					"val2",
					"val3",
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					"val2",
					"val3",
				},
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be any string",
			left: util.JsonObject{
				"stringerino:control": util.JsonObject{
					"must_exist": true,
					"is_string":  true,
				},
			},

			right: util.JsonObject{
				"stringerino": "not equal",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any number",
			left: util.JsonObject{
				"numberino:control": util.JsonObject{
					"must_exist": true,
					"is_number":  true,
				},
			},

			right: util.JsonObject{
				"numberino": util.JsonNumber(99999999),
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be any bool",
			left: util.JsonObject{
				"boolerino:control": util.JsonObject{
					"must_exist": true,
					"is_bool":    true,
				},
			},

			right: util.JsonObject{
				"boolerino": util.JsonBool(false),
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be any array",
			left: util.JsonObject{
				"arrayerino:control": util.JsonObject{
					"must_exist": true,
					"is_array":   true,
				},
			},

			right: util.JsonObject{
				"arrayerino": util.JsonArray(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be an empty object",
			left: util.JsonObject{
				"objecterino": util.JsonObject{},
				"objecterino:control": util.JsonObject{
					"no_extra": true,
				},
			},
			right: util.JsonObject{
				"objecterino": util.JsonObject{"1": 1, "2": 2, "3": 3},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{"objecterino", `extra elements found in object`},
			},
		},

		{
			name: "There should be empty array",
			left: util.JsonObject{
				"arrayerino": util.JsonArray{},
				"arrayerino:control": util.JsonObject{
					"no_extra": true,
				},
			},

			right: util.JsonObject{
				"arrayerino": util.JsonArray{"1", "2", "3"},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{"arrayerino", `extra elements found in array`},
			},
		},

		{
			name: "There should be any object",
			left: util.JsonObject{
				"objecterino:control": util.JsonObject{
					"must_exist": true,
					"is_object":  true,
				},
			},

			right: util.JsonObject{
				"objecterino": util.JsonObject(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "Token match with wrong order",
			left: util.JsonObject{
				"tokens": util.JsonArray{
					util.JsonObject{
						"suggest": "<b>a</b>",
					},
					util.JsonObject{
						"suggest": "<b>a</b>b",
					},
					util.JsonObject{
						"suggest": "<b>a</b>bc",
					},
				},
			},

			right: util.JsonObject{
				"tokens": util.JsonArray{
					util.JsonObject{
						"suggest": "<b>a</b>",
					},
					util.JsonObject{
						"suggest": "<b>a</b>bc",
					},
					util.JsonObject{
						"suggest": "<b>a</b>b",
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be no object",
			left: util.JsonObject{
				"objecterino:control": util.JsonObject{
					"must_not_exist": true,
				},
			},

			right:     util.JsonObject{},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be no object but it exists",
			left: util.JsonObject{
				"objecterino:control": util.JsonObject{
					"must_not_exist": true,
				},
			},

			right: util.JsonObject{
				"objecterino": util.JsonObject(nil),
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "objecterino",
					Message: "actual response[objecterino] was found, but should NOT exist",
				},
			},
		},

		{
			name: "There should be no deeper object but it exists",
			left: util.JsonObject{
				"it": util.JsonArray{
					util.JsonObject{
						"objecterino:control": util.JsonObject{
							"must_not_exist": true,
						},
					},
				},
			},

			right: util.JsonObject{
				"it": util.JsonArray{
					util.JsonObject{
						"objecterino": util.JsonString("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "it[0].objecterino",
					Message: "actual response[objecterino] was found, but should NOT exist",
				},
			},
		},

		{
			name: "There should be no deeper object but it exists2",
			left: util.JsonObject{
				"it": util.JsonArray{
					util.JsonObject{
						"objecterino:control": util.JsonObject{
							"must_not_exist": true,
						},
					},
				},
				"it:control": util.JsonObject{
					"order_matters": true,
				},
			},

			right: util.JsonObject{
				"it": util.JsonArray{
					util.JsonObject{
						"objecterino": util.JsonString("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "it[0].objecterino",
					Message: "actual response[objecterino] was found, but should NOT exist",
				},
			},
		},

		{
			name: "There should be a exact object match",
			left: util.JsonObject{
				"objecterino": util.JsonObject{
					"1": util.JsonNumber(1),
					"2": util.JsonNumber(2),
					"3": util.JsonNumber(3),
				},
				"objecterino:control": util.JsonObject{
					"no_extra": true,
				},
			},

			right: util.JsonObject{
				"objecterino": util.JsonObject{
					"1": util.JsonNumber(1),
					"3": util.JsonNumber(3),
					"2": util.JsonNumber(2),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "There should be a exact object match even if order is mixed",
			left: util.JsonObject{
				"objecterino": util.JsonObject{
					"1": util.JsonNumber(1),
					"2": util.JsonNumber(2),
					"3": util.JsonNumber(3),
				},
				"objecterino:control": util.JsonObject{
					"no_extra": true,
				},
			},

			right: util.JsonObject{
				"objecterino": util.JsonObject{
					"2": util.JsonNumber(2),
					"3": util.JsonNumber(3),
					"1": util.JsonNumber(1),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},

		{
			name: "Exact match is not present",
			left: util.JsonObject{
				"MYobjecterino": util.JsonObject{
					"1": util.JsonNumber(1),
					"2": util.JsonNumber(2),
					"3": util.JsonNumber(3),
				},
				"MYobjecterino:control": util.JsonObject{
					"no_extra": true,
				},
			},

			right: util.JsonObject{
				"MYobjecterino": util.JsonObject{
					"2": util.JsonNumber(2),
					"4": util.JsonNumber(4),
					"1": util.JsonNumber(1),
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "MYobjecterino.3",
					Message: "actual response[3] was not found, but should exists",
				},
				{
					Key:     "MYobjecterino",
					Message: "extra elements found in object",
				},
			},
		},
		{
			name: "Not all contained",
			left: util.JsonObject{
				"array": util.JsonArray{
					"val2",
					"val3",
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					"val1",
					"val2",
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "array[1]",
					Message: "Expected 'val3' != 'val1' Got",
				},
			},
		},
		{
			name: "Wrong order",
			left: util.JsonObject{
				"array": util.JsonArray{
					"val3",
					"val2",
				},
				"array:control": util.JsonObject{
					"order_matters": true,
					"no_extra":      true,
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					"val2",
					"val3",
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "array[0]",
					Message: "Expected 'val3' != 'val2' Got",
				},
				{
					Key:     "array[1]",
					Message: "Expected 'val2' != 'val3' Got",
				},
			},
		},
		{
			name: "Wrong order deeper with map",
			left: util.JsonObject{
				"array": util.JsonObject{
					"inner": util.JsonObject{
						"deeper": util.JsonArray{
							"val4",
							"val5",
						},
						"deeper:control": util.JsonObject{
							"order_matters": true,
						},
					},
				},
			},
			right: util.JsonObject{
				"array": util.JsonObject{
					"inner": util.JsonObject{
						"deeper": util.JsonArray{
							"val5",
							"val4",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "array.inner.deeper[0]",
					Message: "Expected 'val4' != 'val5' Got",
				},
				{
					Key:     "array.inner.deeper[1]",
					Message: "Expected 'val5' != 'val4' Got",
				},
			},
		},
		{
			name: "Right error message for array",
			left: util.JsonObject{
				"body": util.JsonArray{
					util.JsonObject{
						"henk": "denk",
					},
				},
			},
			right: util.JsonObject{
				"body": util.JsonArray{
					util.JsonObject{},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "body[0].henk",
					Message: "actual response[henk] was not found, but should exists",
				},
			},
		},
		/*	{
			name: "Wrong order deeper with arrays",
			left: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						util.JsonArray{
							"val9",
							"val10",
						},
					},
				},
				"array:control": util.JsonObject{
					"order_matters": true,
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						util.JsonArray{
							"val10",
							"val9",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "array",
					Message: "[0][0][0]Expected 'val9' != 'val10' Got",
				},
				{
					Key:     "array",
					Message: "[0][0][1]Expected 'val10' != 'val9' Got",
				},
			},
		},*/
		{
			name: "All fine deeper with arrays",
			left: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						util.JsonArray{
							"val9",
							"val10",
						},
					},
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						util.JsonArray{
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
			left: util.JsonObject{
				"array:control": util.JsonObject{
					"element_count": 3,
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						"val9",
						"val10",
					},
					util.JsonArray{
						"val9",
						"val10",
					},
					util.JsonArray{
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
			left: util.JsonObject{
				"array:control": util.JsonObject{
					"element_count": 2,
				},
				"array": util.JsonArray{
					util.JsonArray{},
				},
			},
			right: util.JsonObject{
				"array": util.JsonArray{
					util.JsonArray{
						util.JsonArray{
							"val9",
							"val10",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []CompareFailure{
				{
					Key:     "array",
					Message: "length of the actual response array '1' != '2' expected length",
				},
			},
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			equal, err := JsonEqual(data.left, data.right, ComparisonContext{})
			if err != nil {
				t.Fatal(err)
			}

			if equal.Equal != data.eEqual {
				t.Errorf("Expected equal '%t' != '%t' Got equal", data.eEqual, equal.Equal)
			}

			if (equal.Failures != nil && data.eFailures == nil) || (equal.Failures == nil && data.eFailures != nil) {
				t.Errorf("Expected Failure '%v' != '%v' Got Failure", data.eFailures, equal.Failures)
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

			test_utils.AssertStringArraysEqualNoOrder(t, wantFailures, haveFailures)

			if t.Failed() {
				t.Log("EXPECTED ", wantFailures)
				t.Log("GOT ", haveFailures)
			}
		})
	}
}
