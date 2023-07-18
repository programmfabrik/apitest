package template

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/test_utils"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/stretchr/testify/assert"
)

func Test_QJson_String(t *testing.T) {
	json := `{
		"foo": "bar"
	}`
	go_test_utils.AssertStringEquals(t, qjson("foo", json), `"bar"`)
}

func Test_QJson_Array(t *testing.T) {
	json := `{
		"foo": [
			"bar",
			1
		]
	}`
	test_utils.AssertJsonStringEquals(t, qjson("foo", json), `[
		"bar",
		1
	]`)
}

func Test_QJson_Object(t *testing.T) {
	json := `{
		"foo": {
			"bar": 1
		}
	}`
	test_utils.AssertJsonStringEquals(t, qjson("foo", json), `{
		"bar": 1
	}`)
}

func TestRowsToMap(t *testing.T) {
	tests := []struct {
		In     []map[string]any
		Key    string
		Value  string
		Out    map[string]any
		ExpErr error
	}{
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "row1b",
				"row2a": "row2b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "row1c",
				"row2a": "row2c",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": map[string]any{
					"column_a": "row1a",
					"column_b": "row1b",
					"column_c": "row1c",
				},
				"row2a": map[string]any{
					"column_a": "row2a",
					"column_b": "row2b",
					"column_c": "row2c",
				},
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "row1b",
				"row2a": "row2b",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "row1b",
				"row2a": "",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "row1b",
				"row3a": "row3b",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out:    map[string]any{},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
				"row1a": "",
				"row2a": "",
				"row3a": "",
			},
			ExpErr: nil,
		},
		{
			In: []map[string]any{
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
			Out: map[string]any{
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
					t.Fatalf("Error: Want '%s' != '%s' Got", v.ExpErr, err)
				}
			} else {
				for k, v := range v.Out {
					mapV, ok := v.(map[string]any)

					if !ok {

						if aOut[k] != v {
							t.Errorf("Value: Want '%s' != '%s' Got", v, aOut[k])
						}
					} else {
						go_test_utils.AssertMapsEqual(t, aOut[k].(map[string]any), mapV)
					}
				}
			}

		})
	}
}

func TestPivot(t *testing.T) {
	data := []map[string]any{
		{
			"key":  "filename",
			"type": "string",
			"1":    "fahrrad",
			"2":    "auto",
			"3":    "dreirad",
		},
		{
			"key":  "wheels",
			"type": "int64",
			"1":    "2",
			"2":    "4",
			"3":    "3",
		},
	}
	exp := []map[string]any{
		{
			"filename": "fahrrad",
			"wheels":   int64(2),
		},
		{
			"filename": "auto",
			"wheels":   int64(4),
		},
		{
			"filename": "dreirad",
			"wheels":   int64(3),
		},
	}
	dataP, err := pivotRows("key", "type", data)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, true, reflect.DeepEqual(dataP, exp)) {
		return
	}
}
