package compare

import (
	"fmt"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/util"
	"github.com/stretchr/testify/assert"
)

var trivialComparerTestData = []struct {
	want  string
	have  string
	match bool
	name  string
	err   error
}{
	{
		`{
			"1": [
				1,
				2,
				3
			]
		}`,
		`{
			"1": [
				1,
				2,
				3
			],
			"2": "a"
		}`,
		true,
		"Left is SubMap of Right",
		nil,
	},
	{
		`{
			"body": [
				{
					"pool": {
						"reference": "system:root"
					}
				}
			],
			"body:control": {
				"no_extra": true
			},
			"statuscode": 200
		}`,
		`{
			"body": [
				{
					"pool": {
						"reference": "system:root"
					}
				},
				{
					"pool2": {
						"reference": "system:root"
					}
				}
			],
			"statuscode": 200
		}`,
		false,
		"Body has more elements",
		nil,
	},
	{
		`[
			1,
			2,
			3
		]`,
		`[
			1,
			2,
			3,
			4
		]`,
		true,
		"Left is SubArray of Right",
		nil,
	},
	{
		`[
			1,
			2,
			3
		]`,
		`[
			1,
			2
		]`,
		false,
		"Left is SuperArray of Right",
		nil,
	},
	{
		`1`,
		`[
			1,
			2,
			3,
			4
		]`,
		false,
		"Different Types Number and Array",
		nil,
	},
	{
		`{
			"1": {
				"1": [
					1,
					2,
					3
				]
			}
		}`,
		`{
			"1": {
				"1": [
					1,
					2
				]
			}
		}`,
		false,
		"Nested List is not contained",
		nil,
	},
	{
		`{
			"1": {
				"1": [
					1,
					2,
					true
				]
			}
		}`,
		`{
			"1": {
				"1": [
					1,
					2,
					true,
					false
				]
			},
			"2": "something"
		}`,
		true,
		"Nested Dicts and Arrays are contained",
		nil,
	},
	{
		`null`,
		`null`,
		true,
		"null should match null",
		nil,
	},
	{
		`[
			"something"
		]`,
		`null`,
		false,
		"null should not match to array",
		nil,
	},
	{
		`[]`,
		`null`,
		false,
		"null should not match to empty array",
		nil,
	},
	{
		`[
			1,
			2,
			{
				"1": null
			}
		]`,
		`[
			1,
			2,
			{
				"1": null,
				"2": "a"
			}
		]`,
		true,
		"nested null should match to nested null",
		nil,
	},
	{
		`"a"`,
		`nil`,
		false,
		"string conversion fails",
		nil,
	},
	{
		`"a"`,
		`"b"`,
		false,
		"string conversion succeeds but comparison fails",
		nil,
	},
	{
		`[
			{
				"event": {
					"object_id": 118
				}
			}
		]`,
		`[
			{
				"event": {
					"_id": 959,
					"basetype": "asset",
					"object_id": 1000832836,
					"object_version": 0,
					"pollable": true,
					"schema": "BASE",
					"timestamp": "2018-11-28T17:37:26+01:00",
					"type": "OBJECT_INDEX"
				}
			},
			{
				"event": {
					"_id": 960,
					"global_object_id": "117@8367e587-f999-4e72-b69d-b5742eb4d5f4",
					"object_id": 117,
					"object_version": 1,
					"objecttype": "pictures",
					"pollable": true,
					"schema": "USER",
					"timestamp": "2018-11-28T17:37:27+01:00",
					"type": "OBJECT_INDEX"
				}
			},
			{
				"event": {
					"_id": 961,
					"global_object_id": "118@8367e587-f999-4e72-b69d-b5742eb4d5f4",
					"object_id": 118,
					"object_version": 1,
					"objecttype": "pictures",
					"pollable": true,
					"schema": "USER",
					"timestamp": "2018-11-28T17:37:27+01:00",
					"type": "OBJECT_INDEX"
				}
			},
			{
				"event": {
					"_id": 962,
					"basetype": "asset",
					"object_id": 1000832836,
					"object_version": 0,
					"pollable": true,
					"schema": "BASE",
					"timestamp": "2018-11-28T17:37:27+01:00",
					"type": "OBJECT_INDEX"
				}
			},
			{
				"event": {
					"_id": 963,
					"basetype": "asset",
					"object_id": 1000832835,
					"object_version": 0,
					"pollable": true,
					"schema": "BASE",
					"timestamp": "2018-11-28T17:37:27+01:00",
					"type": "OBJECT_INDEX"
				}
			}
		]`,
		true,
		"Match events",
		nil,
	},
	{

		`{
			"body": [
				{
					"henk": "denk"
				}
			]
		}`,
		`{
			"body": [
				{
				}
			]
		}`,
		false,
		"ticket #51342. Error msg",
		fmt.Errorf("[body[0].henk] was not found, but should exist"),
	},
	{

		`{
			"body": [
				{
					"henk": null
				}
			]
		}`,
		`{
			"body": [
				{
				}
			]
		}`,
		false,
		"ticket #52417. check value null. Null not found",
		fmt.Errorf("[body[0].henk] was not found, but should exist"),
	},
	{

		`{
			"body": [
				{
					"henk": null
				}
			]
		}`,
		`{
			"body": [
				{
					"henk": "2"
				}
			]
		}`,
		false,
		"ticket #52417. check value null. Other value than null",
		fmt.Errorf("[body[0].henk] the type of the expected response is invalid. Got 'string', expected '<nil>'"),
	},
	{

		`{
			"body": [
				{
					"henk": null
				}
			]
		}`,
		`{
			"body": [
				{
					"henk": null
				}
			]
		}`,
		true,
		"ticket #52417. check value null. Null found",
		nil,
	},
}

func TestTrivialJsonComparer(t *testing.T) {
	var json1, json2 any
	for _, td := range trivialComparerTestData {
		t.Run(td.name, func(t *testing.T) {
			util.UnmarshalWithNumber([]byte(td.want), &json1)
			util.UnmarshalWithNumber([]byte(td.have), &json2)
			tjcMatch, err := JsonEqual(json1, json2, ComparisonContext{})
			if err != nil {
				t.Fatal("Error occurred: ", err)
			}
			if td.match != tjcMatch.Equal {
				t.Errorf("Got %t, expected %t", tjcMatch.Equal, td.match)
			}

			if td.err != nil {
				if len(tjcMatch.Failures) != 1 || td.err.Error() != tjcMatch.Failures[0].String() {
					t.Errorf("Error missmatch. Got '%s', epected '%s'", tjcMatch.Failures[0].String(), td.err)
				}
			}
		})
	}
}

func TestJsonNumberEq(t *testing.T) {
	if !assert.Equal(t, true, jsonNumberEq("200", "200")) {
		return
	}
	if !assert.Equal(t, false, jsonNumberEq("-9223372036854775808", "-9223372036854775809")) {
		return
	}
	if !assert.Equal(t, false, jsonNumberEq("-9223372036854775809", "-9223372036854775808")) {
		return
	}
	if !assert.Equal(t, true, jsonNumberEq("1e10", "10000000000")) {
		return
	}
	// Although this is the same value, we cannot say its equal
	if !assert.Equal(t, true, jsonNumberEq("-9.223372036854775808e+18", "-9223372036854775808")) {
		return
	}
	if !assert.Equal(t, true, jsonNumberEq("-9.223372036854775808e+18", "-9.223372036854775808e+18")) {
		return
	}
	if !assert.Equal(t, false, jsonNumberEq("-9.223372036854775809e+18", "-9223372036854775809")) {
		return
	}
}
