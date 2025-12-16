package util

import (
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
)

func TestRemoveFromJsonArray(t *testing.T) {
	input := []any{
		jsutil.String("0"),
		jsutil.String("1"),
		jsutil.String("2"),
		jsutil.String("3"),
		jsutil.String("4"),
		jsutil.String("5"),
	}

	output := removeFromJsonArray(input, 2)
	if len(output) != 5 || output[2] != jsutil.String("3") {
		t.Errorf("Wrong slice removal: %s", output)
	}

	output2 := removeFromJsonArray(input, 5)
	if len(output2) != 5 || output2[4] != jsutil.String("4") {
		t.Errorf("Wrong slice removal: %s", output2)
	}

	output3 := removeFromJsonArray(input, 0)
	if len(output3) != 5 || output3[4] != jsutil.String("5") {
		t.Errorf("Wrong slice removal: %s", output3)
	}
}
