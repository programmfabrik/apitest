package util

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func RemoveFromJsonArray(input []GenericJson, removeIndex int) (output []GenericJson) {
	output = make([]GenericJson, len(input))
	copy(output, input)

	// Remove the element at index i from a.
	copy(output[removeIndex:], input[removeIndex+1:]) // Shift a[i+1:] left one index.
	output[len(output)-1] = nil                       // Erase last element (write zero value).
	output = output[:len(output)-1]                   // Truncate slice.

	return output
}
