package util

import "testing"

func TestRemoveFromJSONArray(t *testing.T) {
	input := []interface{}{JSONString("0"), JSONString("1"), JSONString("2"), JSONString("3"), JSONString("4"), JSONString("5")}

	output := RemoveFromJSONArray(input, 2)
	if len(output) != 5 || output[2] != JSONString("3") {
		t.Errorf("Wrong slice removal: %s", output)
	}

	output2 := RemoveFromJSONArray(input, 5)
	if len(output2) != 5 || output2[4] != JSONString("4") {
		t.Errorf("Wrong slice removal: %s", output2)
	}

	output3 := RemoveFromJSONArray(input, 0)
	if len(output3) != 5 || output3[4] != JSONString("5") {
		t.Errorf("Wrong slice removal: %s", output3)
	}
}
