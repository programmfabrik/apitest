package template

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/stretchr/testify/assert"
)

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

func TestExtendForJsonNumber(t *testing.T) {
	store := datastore.NewStore(false)
	loader := NewLoader(store)

	type testCase struct {
		tmpl, exp string
	}
	for _, c := range []testCase{
		{
			tmpl: `{{ if eq "1" "1" }} 1==1 {{ else }} 1!=1 {{ end }}`,
			exp:  `1==1`,
		},
		{
			tmpl: `{{ if ne 2 3 }}2!=3{{ else }}2==3{{ end }}`,
			exp:  `2!=3`,
		},
		{
			tmpl: `{{ add 4 5 }}`,
			exp:  `9`,
		},
		{
			tmpl: `{{ sub 6 1 }}`,
			exp:  `5`,
		},
		{
			tmpl: `{{ if lt 7 8 }} 7<8 {{ else }} 7>=8 {{ end }}`,
			exp:  `7<8`,
		},
		{
			tmpl: `{{ if gt 10 9 }} 10>9 {{ else }} 9<=10 {{ end }}`,
			exp:  `10>9`,
		},
	} {
		res, err := loader.Render([]byte(c.tmpl), "", nil)
		go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
		if !assert.Equal(t, strings.TrimSpace(c.exp), strings.TrimSpace(string(res))) {
			return
		}
	}
}
