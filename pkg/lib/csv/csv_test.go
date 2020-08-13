package csv

import (
	"encoding/json"
	"testing"

	go_test_utils "github.com/programmfabrik/go-test-utils"
)

// TestCSVToMap_StringArray checks if trailing spaces are preserved in CSV
// values of type "string,array".
func TestCSVToMap_StringArray(t *testing.T) {
	csvData := []byte(`list
"string,array"
"No surrounding,spaces"
" Leading,space"
"Trailing ,space"
" Surrounding ,spaces"`)

	jsnData := []byte(`[
{ "list": ["No surrounding","spaces"] },
{ "list": [" Leading","space"] },
{ "list": ["Trailing ","space"] },
{ "list": [" Surrounding ","spaces"] }
]`)

	got, err := CSVToMap(csvData, ',')
	if err != nil {
		t.Fatal(err)
	}

	expect := make([]map[string][]string, 0)
	if err := json.Unmarshal(jsnData, &expect); err != nil {
		t.Fatal(err)
	}

	for i := range expect {
		a := got[i]["list"].([]string)
		b := expect[i]["list"]

		go_test_utils.AssertStringArraysEqualNoOrder(t, a, b)
	}
}
