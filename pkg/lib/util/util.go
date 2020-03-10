package util

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func RemoveFromJSONArray(input []interface{}, removeIndex int) (output []interface{}) {
	output = make([]interface{}, len(input))
	copy(output, input)

	// Remove the element at index i from a.
	copy(output[removeIndex:], input[removeIndex+1:]) // Shift a[i+1:] left one index.
	output[len(output)-1] = nil                       // Erase last element (write zero value).
	output = output[:len(output)-1]                   // Truncate slice.

	return output
}

func GetStringFromInterface(queryParam interface{}) (string, error) {
	switch t := queryParam.(type) {
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case int:
		return fmt.Sprintf("%d", t), nil
	default:
		jsonVal, err := json.Marshal(t)
		return string(jsonVal), err
	}
}
