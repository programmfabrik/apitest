package compare

import (
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"github.com/programmfabrik/fylr-apitest/lib/util"
)

var trivialComparerTestData = []struct {
	in1   string
	in2   string
	match bool
	name  string
}{
	{
		`{"1":[1,2,3]}`,
		`{"2":"a","1":[1,2,3]}`,
		true,
		"Left is SubMap of Right"},
	{
		`[1, 2, 3]`,
		`[1, 2, 3, 4]`,
		true,
		"Left is SubArray of Right",
	},
	{
		`[1, 2, 3]`,
		`[1, 2]`,
		false,
		"Left is SuperArray of Right",
	},
	{
		`1`,
		`[1, 2, 3, 4]`,
		false,
		"Different Types Number and Array",
	},
	{
		`{"1": {"1": [1, 2, 3]}}`,
		`{"1": {"1": [1, 2]}}`,
		false,
		"Nested List is not contained",
	},
	{
		`{"1": {"1": [1, 2, true]}}`,
		`{"2": "something", "1": {"1": [1, 2, true, false]}}`,
		true,
		"Nested Dicts and Arrays are contained",
	},
	{
		`null`,
		`null`,
		true,
		"null should match null",
	},
	{
		`["something"]`,
		`null`,
		false,
		"null should not match to array",
	},
	{
		`[]`,
		`null`,
		false,
		"null should not match to empty array",
	},
	{
		`[1, 2, {"1": null}]`,
		`[1, 2, {"1": null, "2": "a"}]`,
		true,
		"nested null should match to nested null",
	},
	{
		`"a"`,
		`nil`,
		false,
		"string conversion fails",
	},
	{
		`"a"`,
		`"b"`,
		false,
		"string conversion succeeds but comparison fails",
	},
	{
		`[{"event":{"object_id":118}}] `,
		`[
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 959,
            "object_version": 0,
            "object_id": 1000832836,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:26+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 960,
            "object_version": 1,
            "object_id": 117,
            "schema": "USER",
            "objecttype": "pictures",
            "global_object_id": "117@8367e587-f999-4e72-b69d-b5742eb4d5f4",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 961,
            "object_version": 1,
            "object_id": 118,
            "schema": "USER",
            "objecttype": "pictures",
            "global_object_id": "118@8367e587-f999-4e72-b69d-b5742eb4d5f4",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 962,
            "object_version": 0,
            "object_id": 1000832836,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 963,
            "object_version": 0,
            "object_id": 1000832835,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    }
]
`,
		true,
		"Match events",
	},
}

func TestTrivialJsonComparer(t *testing.T) {
	var json1, json2 util.GenericJson
	for _, td := range trivialComparerTestData {
		t.Run(td.name, func(t *testing.T) {
			cjson.Unmarshal([]byte(td.in1), &json1)
			cjson.Unmarshal([]byte(td.in2), &json2)
			tjcMatch, err := JsonEqual(json1, json2, ComparisonContext{})
			if err != nil {
				t.Errorf("error occured")
			}
			if !(td.match == tjcMatch.Equal) {
				t.Errorf("got %t, want %t", tjcMatch.Equal, td.match)
			}

		})
	}
}
