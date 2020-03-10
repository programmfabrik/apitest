package template

import (
	"fmt"
	"testing"

	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func Test_QJSON_String(t *testing.T) {
	json := `{"foo": "bar"}`
	go_test_utils.AssertStringEquals(t, qjson("foo", json), "\"bar\"")
}

func Test_QJSON_Array(t *testing.T) {
	json := `{"foo": ["bar", 1]}`
	go_test_utils.AssertStringEquals(t, qjson("foo", json), "[\"bar\", 1]")
}

func Test_QJSON_Object(t *testing.T) {
	json := `{"foo": {"bar": 1}}`
	go_test_utils.AssertStringEquals(t, qjson("foo", json), "{\"bar\": 1}")
}

func TestRowsToMap(t *testing.T) {
	tests := []struct {
		In     []map[string]interface{}
		Key    string
		Value  string
		Out    map[string]interface{}
		ExpErr error
	}{
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
					"column_c": "row1c",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
					"column_c": "row2c",
				},
			},
			Key:   "column_a",
			Value: "column_b",
			Out: map[string]interface{}{
				"row1a": "row1b",
				"row2a": "row2b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
					"column_c": "row1c",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
					"column_c": "row2c",
				},
			},
			Key:   "column_a",
			Value: "column_c",
			Out: map[string]interface{}{
				"row1a": "row1c",
				"row2a": "row2c",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
					"column_c": "row1c",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
					"column_c": "row2c",
				},
			},
			Key:   "column_a",
			Value: "",
			Out: map[string]interface{}{
				"row1a": map[string]interface{}{
					"column_a": "row1a",
					"column_b": "row1b",
					"column_c": "row1c",
				},
				"row2a": map[string]interface{}{
					"column_a": "row2a",
					"column_b": "row2b",
					"column_c": "row2c",
				},
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
				},
				{
					"column_a": "row3a",
					"column_b": "row3b",
				},
			},
			Key:   "column_a",
			Value: "column_b",
			Out: map[string]interface{}{
				"row1a": "row1b",
				"row2a": "row2b",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "row2a",
				},
				{
					"column_a": "row3a",
					"column_b": "row3b",
				},
			},
			Key:   "column_a",
			Value: "column_b",
			Out: map[string]interface{}{
				"row1a": "row1b",
				"row2a": "",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "",
					"column_b": "row2b",
				},
				{
					"column_a": "row3a",
					"column_b": "row3b",
				},
			},
			Key:   "column_a",
			Value: "column_b",
			Out: map[string]interface{}{
				"row1a": "row1b",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
				},
				{
					"column_a": "row3a",
					"column_b": "row3b",
				},
			},
			Key:    "column_ZZ",
			Value:  "column_b",
			Out:    map[string]interface{}{},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
				},
				{
					"column_a": "row3a",
					"column_b": "row3b",
				},
			},
			Key:   "column_a",
			Value: "column_ZZ",
			Out: map[string]interface{}{
				"row1a": "",
				"row2a": "",
				"row3a": "",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]interface{}{
				{
					"column_a": "row1a",
					"column_b": "row1b",
				},
				{
					"column_a": "row2a",
					"column_b": "row2b",
				},
				{
					"column_a": 22,
					"column_b": "row3b",
				},
			},
			Key:   "column_a",
			Value: "column_ZZ",
			Out: map[string]interface{}{
				"row1a": "",
				"row2a": "",
				"row3a": "",
			},
			ExpErr: fmt.Errorf("'22' must be string, as it functions as map index"),
		},
	}

	for k, v := range tests {
		t.Run(fmt.Sprintf("case_%d", k), func(t *testing.T) {

			aOut, err := rowsToMap(v.Key, v.Value, v.In)

			if err != nil {

				if v.ExpErr == nil || err.Error() != v.ExpErr.Error() {
					t.Fatalf("Error: got '%s', expected '%s'", err, v.ExpErr)
				}
			} else {
				for k, v := range v.Out {
					mapV, ok := v.(map[string]interface{})

					if !ok {

						if aOut[k] != v {
							t.Errorf("Value: got '%s', expected '%s'", aOut[k], v)
						}
					} else {
						go_test_utils.AssertMapsEqual(t, aOut[k].(map[string]interface{}), mapV)
					}
				}
			}

		})
	}
}
