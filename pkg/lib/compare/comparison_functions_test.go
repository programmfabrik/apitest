package compare

import (
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/yudai/pp"
)

func TestComparison(t *testing.T) {
	testData := []struct {
		name      string
		left      jsutil.Object
		right     jsutil.Object
		eEqual    bool
		eFailures []compareFailure
	}{
		{
			name: "Should be equal",
			left: jsutil.Object{
				"array": jsutil.Array{
					"val2",
					"val3",
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					"val2",
					"val3",
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any string",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"must_exist": true,
					"is_string":  true,
				},
			},

			right: jsutil.Object{
				"stringerino": "not equal",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String matches regex",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"match":     "\\d+\\..+",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String must not match regex",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"not_match": "\\d+\\..+",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "matches regex \"\\\\d+\\\\..+\" but should not match",
				},
			},
		},
		{
			name: "String must not match regex",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"not_match": "\\d+\\..+",
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "matches regex \"\\\\d+\\\\..+\" but should not match",
				},
			},
		},
		{
			name: "String does not match regex",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"match":     "\\d+\\.\\d+",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "xyz-456",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "string \"xyz-456\" does not match regex \"\\\\d+\\\\.\\\\d+\"",
				},
			},
		},
		{
			name: "String match with invalid regex (must fail)",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"match":     ".+[",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "could not match regex \".+[\": error parsing regexp: missing closing ]: `[`",
				},
			},
		},
		{
			name: "String starts with another string",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"starts_with": "123.",
					"is_string":   true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String does not start with another string",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"starts_with": "789.",
					"is_string":   true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "does not start with '789.'",
				},
			},
		},
		{
			name: "String ends with another string",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"ends_with": ".abc",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "String does not end with another string",
			left: jsutil.Object{
				"stringerino:control": jsutil.Object{
					"ends_with": ".xyz",
					"is_string": true,
				},
			},
			right: jsutil.Object{
				"stringerino": "123.abc",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "stringerino",
					Message: "does not end with '.xyz'",
				},
			},
		},
		{
			name: "There should be any number",
			left: jsutil.Object{
				"numberino:control": jsutil.Object{
					"must_exist": true,
					"is_number":  true,
				},
			},
			right: jsutil.Object{
				"numberino": jsutil.Number("99999999"),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any bool",
			left: jsutil.Object{
				"boolerino:control": jsutil.Object{
					"must_exist": true,
					"is_bool":    true,
				},
			},
			right: jsutil.Object{
				"boolerino": jsutil.Bool(false),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be any array",
			left: jsutil.Object{
				"arrayerino:control": jsutil.Object{
					"must_exist": true,
					"is_array":   true,
				},
			},
			right: jsutil.Object{
				"arrayerino": jsutil.Array(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be an empty object",
			left: jsutil.Object{
				"objecterino": jsutil.Object{},
				"objecterino:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"objecterino": jsutil.Object{"1": 1, "2": 2, "3": 3},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{"objecterino", `extra elements found in object`},
			},
		},
		{
			name: "There should be empty array",
			left: jsutil.Object{
				"arrayerino": jsutil.Array{},
				"arrayerino:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"arrayerino": jsutil.Array{"1", "2", "3"},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{"arrayerino", `extra elements found in array`},
			},
		},
		{
			name: "There should be any object",
			left: jsutil.Object{
				"objecterino:control": jsutil.Object{
					"must_exist": true,
					"is_object":  true,
				},
			},
			right: jsutil.Object{
				"objecterino": jsutil.Object(nil),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Token match with wrong order",
			left: jsutil.Object{
				"tokens": jsutil.Array{
					jsutil.Object{
						"suggest": "<b>a</b>",
					},
					jsutil.Object{
						"suggest": "<b>a</b>b",
					},
					jsutil.Object{
						"suggest": "<b>a</b>bc",
					},
				},
			},
			right: jsutil.Object{
				"tokens": jsutil.Array{
					jsutil.Object{
						"suggest": "<b>a</b>",
					},
					jsutil.Object{
						"suggest": "<b>a</b>bc",
					},
					jsutil.Object{
						"suggest": "<b>a</b>b",
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be no object",
			left: jsutil.Object{
				"objecterino:control": jsutil.Object{
					"must_not_exist": true,
				},
			},
			right:     jsutil.Object{},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be no object but it exists",
			left: jsutil.Object{
				"objecterino:control": jsutil.Object{
					"must_not_exist": true,
				},
			},
			right: jsutil.Object{
				"objecterino": jsutil.Object(nil),
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be no deeper object but it exists",
			left: jsutil.Object{
				"it": jsutil.Array{
					jsutil.Object{
						"objecterino:control": jsutil.Object{
							"must_not_exist": true,
						},
					},
				},
			},
			right: jsutil.Object{
				"it": jsutil.Array{
					jsutil.Object{
						"objecterino": jsutil.String("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "it[0].objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be no deeper object but it exists2",
			left: jsutil.Object{
				"it": jsutil.Array{
					jsutil.Object{
						"objecterino:control": jsutil.Object{
							"must_not_exist": true,
						},
					},
				},
			},
			right: jsutil.Object{
				"it": jsutil.Array{
					jsutil.Object{
						"objecterino": jsutil.String("I AM HERE"),
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "it[0].objecterino",
					Message: "was found, but should NOT exist",
				},
			},
		},
		{
			name: "There should be a exact object match",
			left: jsutil.Object{
				"objecterino": jsutil.Object{
					"1": jsutil.Number("1"),
					"2": jsutil.Number("2"),
					"3": jsutil.Number("3"),
				},
				"objecterino:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"objecterino": jsutil.Object{
					"1": jsutil.Number("1"),
					"3": jsutil.Number("3"),
					"2": jsutil.Number("2"),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "There should be a exact object match even if order is mixed",
			left: jsutil.Object{
				"objecterino": jsutil.Object{
					"1": jsutil.Number("1"),
					"2": jsutil.Number("2"),
					"3": jsutil.Number("3"),
				},
				"objecterino:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"objecterino": jsutil.Object{
					"2": jsutil.Number("2"),
					"3": jsutil.Number("3"),
					"1": jsutil.Number("1"),
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Exact match is not present",
			left: jsutil.Object{
				"MYobjecterino": jsutil.Object{
					"1": jsutil.Number("1"),
					"2": jsutil.Number("2"),
					"3": jsutil.Number("3"),
				},
				"MYobjecterino:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"MYobjecterino": jsutil.Object{
					"2": jsutil.Number("2"),
					"4": jsutil.Number("4"),
					"1": jsutil.Number("1"),
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
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
			left: jsutil.Object{
				"array": jsutil.Array{
					"val2",
					"val3",
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					"val1",
					"val2",
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "array[1]",
					Message: "Got 'val1', expected 'val3'",
				},
			},
		},
		{
			name: "Wrong order",
			left: jsutil.Object{
				"array": jsutil.Array{
					"val3",
					"val2",
				},
				"array:control": jsutil.Object{
					"order_matters": true,
					"no_extra":      true,
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					"val2",
					"val3",
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "array[1]",
					Message: "element \"val2\" not found in array in proper order",
				},
				{
					Key:     "array",
					Message: "extra elements found in array",
				},
			},
		},
		{
			name: "Wrong order deeper with map",
			left: jsutil.Object{
				"array": jsutil.Object{
					"inner": jsutil.Object{
						"deeper": jsutil.Array{
							"val4",
							"val5",
						},
						"deeper:control": jsutil.Object{
							"order_matters": true,
						},
					},
				},
			},
			right: jsutil.Object{
				"array": jsutil.Object{
					"inner": jsutil.Object{
						"deeper": jsutil.Array{
							"val5",
							"val4",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "array.inner.deeper[1]",
					Message: "element \"val5\" not found in array in proper order",
				},
			},
		},
		{
			name: "Right error message for array",
			left: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"henk": "denk",
					},
				},
			},
			right: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "body[0].henk",
					Message: "was not found, but should exist",
				},
			},
		},
		/*	{
			name: "Wrong order deeper with arrays",
			left: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						jsutil.Array{
							"val9",
							"val10",
						},
					},
				},
				"array:control": jsutil.Object{
					"order_matters": true,
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						jsutil.Array{
							"val10",
							"val9",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
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
			left: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						jsutil.Array{
							"val9",
							"val10",
						},
					},
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						jsutil.Array{
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
			left: jsutil.Object{
				"array:control": jsutil.Object{
					"element_count": 3,
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						"val9",
						"val10",
					},
					jsutil.Array{
						"val9",
						"val10",
					},
					jsutil.Array{
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
			left: jsutil.Object{
				"array:control": jsutil.Object{
					"element_count": 2,
				},
				"array": jsutil.Array{
					jsutil.Array{},
				},
			},
			right: jsutil.Object{
				"array": jsutil.Array{
					jsutil.Array{
						jsutil.Array{
							"val10",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "array",
					Message: "length of the actual response array 1 != 2 expected length",
				},
			},
		},
		{
			name: "Check controls in array (1 element)",
			left: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference:control": jsutil.Object{
								"is_number": true,
							},
						},
					},
				},
			},
			right: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": "system:root",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "body[0].pool.reference",
					Message: "should be 'JsonNumber' or 'Number' but is 'String'",
				},
			},
		},
		{
			name: "Check controls in array (more elements)",
			left: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference:control": jsutil.Object{
								"is_number": true,
							},
						},
					},
					jsutil.Object{
						"pool": jsutil.Object{
							"reference:control": jsutil.Object{
								"is_number": true,
							},
						},
					},
				},
			},
			right: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": "system:root",
						},
					},
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": 123,
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "body[1].pool.reference",
					Message: "should be 'JsonNumber' or 'Number' but is 'String'",
				},
			},
		},
		{
			name: "Check controls in array (more elements, different order)",
			left: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference:control": jsutil.Object{
								"is_number": true,
							},
						},
					},
					jsutil.Object{
						"pool": jsutil.Object{
							"reference:control": jsutil.Object{
								"is_string": true,
							},
						},
					},
				},
			},
			right: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": "system:root",
						},
					},
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": 123,
						},
					},
				},
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check body no extra",
			left: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": "system:root",
						},
					},
				},
				"body:control": jsutil.Object{
					"no_extra": true,
				},
			},
			right: jsutil.Object{
				"body": jsutil.Array{
					jsutil.Object{
						"pool": jsutil.Object{
							"reference":  "system:root",
							"reference2": "system:root",
						},
					},
					jsutil.Object{
						"pool": jsutil.Object{
							"reference": "system:root",
						},
					},
				},
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "body",
					Message: "extra elements found in array",
				},
			},
		},
		{
			name: "check control not_equal (different types number, string)",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": "right",
				},
			},
			right: jsutil.Object{
				"v": 123.456,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "check control not_equal (different types number, array)",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": jsutil.Array([]any{"right"}),
				},
			},
			right: jsutil.Object{
				"v": 123.456,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "check control not_equal (different types string, number)",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": 123.45,
				},
			},
			right: jsutil.Object{
				"v": "left",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "check control not_equal (different types bool, number)",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": 456.789,
				},
			},
			right: jsutil.Object{
				"v": true,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "check control not_equal (different types number, bool)",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": true,
				},
			},
			right: jsutil.Object{
				"v": 789.0001,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal with null and null",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": nil,
				},
			},
			right: jsutil.Object{
				"v": nil,
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "v",
					Message: "is null",
				},
			},
		},
		{
			name: "Check not_equal with null and string",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": nil,
				},
			},
			right: jsutil.Object{
				"v": "not null",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal with string and null",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": "not null",
				},
			},
			right: jsutil.Object{
				"v": nil,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal with null and array",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": nil,
				},
			},
			right: jsutil.Object{
				"v": jsutil.Array([]any{"not null"}),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal with array and null",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": jsutil.Array([]any{"not null"}),
				},
			},
			right: jsutil.Object{
				"v": nil,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal: string with different value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": "left",
				},
			},
			right: jsutil.Object{
				"v": "right",
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal: string with same value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": "left",
				},
			},
			right: jsutil.Object{
				"v": "left",
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "v",
					Message: "is equal to String 'left', should not be equal",
				},
			},
		},
		{
			name: "Check not_equal: array with different value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": jsutil.Array([]any{"left", "right"}),
				},
			},
			right: jsutil.Object{
				"v": jsutil.Array([]any{"right", "left"}),
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal: array with same value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": jsutil.Array([]any{"left", "right"}),
				},
			},
			right: jsutil.Object{
				"v": jsutil.Array([]any{"left", "right"}),
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "v",
					Message: `is equal to Array ["left","right"], should not be equal`,
				},
			},
		},
		{
			name: "Check not_equal: number with different value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": 123.45,
				},
			},
			right: jsutil.Object{
				"v": 6.789,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal: number with same value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": 0.111,
				},
			},
			right: jsutil.Object{
				"v": 0.111,
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "v",
					Message: "is equal to Number 0.111, should not be equal",
				},
			},
		},
		{
			name: "Check not_equal: boolean with different value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": false,
				},
			},
			right: jsutil.Object{
				"v": true,
			},
			eEqual:    true,
			eFailures: nil,
		},
		{
			name: "Check not_equal: boolean with same value",
			left: jsutil.Object{
				"v:control": jsutil.Object{
					"not_equal": false,
				},
			},
			right: jsutil.Object{
				"v": false,
			},
			eEqual: false,
			eFailures: []compareFailure{
				{
					Key:     "v",
					Message: "is equal to Bool false, should not be equal",
				},
			},
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			equal, err := JsonEqual(data.left, data.right, ComparisonContext{})
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

			if equal.Equal != data.eEqual {
				t.Errorf("Expected '%t', got '%t'", data.eEqual, equal.Equal)
				return
			}

			if (equal.Failures != nil && data.eFailures == nil) || (equal.Failures == nil && data.eFailures != nil) {
				t.Errorf("Expected Failure '%v' != '%v' Got Failure", data.eFailures, equal.Failures)
				return
			}

			wantFailures := []string{}
			for _, v := range data.eFailures {
				wantFailures = append(wantFailures, v.String())
			}
			haveFailures := []string{}
			for _, v := range equal.Failures {
				haveFailures = append(haveFailures, v.String())
			}

			go_test_utils.AssertStringArraysEqualNoOrder(t, wantFailures, haveFailures)

			if t.Failed() {
				pp.Println(equal)
				t.Log("EXPECTED ", wantFailures)
				t.Log("GOT      ", haveFailures)
			}
		})
	}
}
