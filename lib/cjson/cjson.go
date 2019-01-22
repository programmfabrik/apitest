package cjson

import (
	"encoding/json"
	"fmt"
	"regexp"
)

func Unmarshal(input []byte, output interface{}) error {
	var commentRegex = regexp.MustCompile(`(?m)^(.*?)(#|//).*$`)
	inputNoComments := []byte(commentRegex.ReplaceAllString(string(input), `$1`))

	// unmarshal into object
	err := json.Unmarshal(inputNoComments, output)
	if err != nil {
		return getIndepthJsonError(inputNoComments, err)
	}
	return nil
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func getIndepthJsonError(input []byte, inputError error) (err error) {

	err = inputError

	if jsonError, ok := inputError.(*json.SyntaxError); ok {
		line, character, lcErr := lineAndCharacter(string(input), int(jsonError.Offset))
		if lcErr != nil {
			err = jsonError
			return
		}

		err = fmt.Errorf("Cannot parse JSON '%s' schema due to a syntax error at line %d, character %d: %v",
			string(input), line, character, jsonError.Error())
		return
	}

	if jsonError, ok := inputError.(*json.UnmarshalTypeError); ok {
		line, character, lcErr := lineAndCharacter(string(input), int(jsonError.Offset))
		if lcErr != nil {
			err = jsonError
			return
		}

		return fmt.Errorf(`In JSON '%s', the type '%v' cannot be converted into the Go '%v' type on struct '%s', field '%v'. See input file line %d, character %d`,
			string(input), jsonError.Value, jsonError.Type.Name(), jsonError.Struct, jsonError.Field, line, character)
	}

	return
}

func lineAndCharacter(input string, offset int) (line int, character int, err error) {
	if offset > len(input) || offset < 0 {
		return 0, 0, fmt.Errorf("Couldn't find offset %d within the input.", offset)
	}
	//humans count line from 1
	line = 1
	for _, b := range input[:offset] {
		if b == rune('\n') {
			line++
			character = 0
		} else {
			character++
		}
	}

	return line, character, nil
}
