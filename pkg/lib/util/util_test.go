package util

import "testing"

func TestRemoveFromJsonArray(t *testing.T) {
	input := []any{JsonString("0"), JsonString("1"), JsonString("2"), JsonString("3"), JsonString("4"), JsonString("5")}

	output := removeFromJsonArray(input, 2)
	if len(output) != 5 || output[2] != JsonString("3") {
		t.Errorf("Wrong slice removal: %s", output)
	}

	output2 := removeFromJsonArray(input, 5)
	if len(output2) != 5 || output2[4] != JsonString("4") {
		t.Errorf("Wrong slice removal: %s", output2)
	}

	output3 := removeFromJsonArray(input, 0)
	if len(output3) != 5 || output3[4] != JsonString("5") {
		t.Errorf("Wrong slice removal: %s", output3)
	}
}
