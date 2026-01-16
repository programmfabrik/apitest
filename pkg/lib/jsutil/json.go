package jsutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/programmfabrik/golib"
	"github.com/tidwall/jsonc"
)

type (
	Object     = map[string]any
	Array      = []any
	String     = string
	Number     = json.Number
	Bool       = bool
	RawMessage = json.RawMessage
)

var (
	coloredError      bool
	cjsonCommentRegex = regexp.MustCompile(`(?m)^[\t ]*#.*$`)
)

func init() {
	coloredError = true
}

// NumberEqual is comparing ints, floats or strings of the number. It fails to
// compare different formats, 1e10 != 10000000000, although it is the same mathematical value.
func NumberEqual(numberExp, numberGot Number) (eq bool) {

	expInt, expIntErr := numberExp.Int64()
	gotInt, gotIntErr := numberGot.Int64()
	expFloat, expFloatErr := numberExp.Float64()
	gotFloat, gotFloatErr := numberGot.Float64()

	var cmp string
	_ = cmp

	if expIntErr == nil && gotIntErr == nil {
		cmp = "int"
	} else if expFloatErr == nil && gotFloatErr == nil {
		cmp = "float"
	} else {
		cmp = "string"
	}

	// if any of the interpretations is out of range, we compare by string
	for _, e := range []error{
		expIntErr, gotIntErr, expFloatErr, gotFloatErr,
	} {
		if e == nil {
			continue
		}
		if strings.Contains(e.Error(), "range") {
			cmp = "string"
			break
		}
	}

	switch cmp {
	case "int":
		eq = expInt == gotInt
	case "float":
		eq = expFloat == gotFloat
	case "string":
		eq = numberExp == numberGot
	}

	return eq

}

// Marshal converts the given interface into json bytes
func Marshal(v any) (data []byte, err error) {
	return golib.JsonBytes(v)
}

// Encode marshals the given interface and writes the json bytes to the given writer
func Encode(w io.Writer, v any) (err error) {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalString is a wrapper for Unmarshal for string input
func UnmarshalString(input string, output any) (err error) {
	return Unmarshal([]byte(input), output)
}

// Unmarshal decodes the input bytes into the output if it is valid cjson
func Unmarshal(input []byte, output any) (err error) {
	// Remove # comments from template
	tmplBytes := cjsonCommentRegex.ReplaceAll(input, []byte{})

	// Remove //, /* comments plus tailing commas
	tmplBytes = jsonc.ToJSON(tmplBytes)

	dec := json.NewDecoder(bytes.NewReader(tmplBytes))
	dec.DisallowUnknownFields()
	dec.UseNumber()

	// unmarshal into object
	err = dec.Decode(output)
	if err != nil {
		return getIndepthJsonError(tmplBytes, err)
	}
	return nil
}

func getIndepthJsonError(input []byte, inputError error) (err error) {
	var (
		syntaxError        *json.SyntaxError
		unmarshalTypeError *json.UnmarshalTypeError
		ok                 bool
		line, character    int
		lcErr              error
	)

	err = inputError

	syntaxError, ok = inputError.(*json.SyntaxError)
	if ok {
		line, character, lcErr = lineAndCharacter(string(input), int(syntaxError.Offset))
		if lcErr != nil {
			err = syntaxError
			return
		}

		err = fmt.Errorf(
			"Cannot parse JSON '%s' schema due to a syntax error at line %d, character %d: %v",
			getErrorJsonWithLineNumbers(string(input), line),
			line,
			character,
			syntaxError.Error(),
		)
		return
	}

	unmarshalTypeError, ok = inputError.(*json.UnmarshalTypeError)
	if ok {
		line, character, lcErr = lineAndCharacter(string(input), int(unmarshalTypeError.Offset))
		if lcErr != nil {
			err = unmarshalTypeError
			return
		}

		return fmt.Errorf(
			`In JSON '%s', the type '%v' cannot be converted into the Go '%v' type on struct '%s', field '%v'. See input file line %d, character %d`,
			getErrorJsonWithLineNumbers(string(input), line),
			unmarshalTypeError.Value,
			unmarshalTypeError.Type.Name(),
			unmarshalTypeError.Struct,
			unmarshalTypeError.Field,
			line,
			character,
		)
	}

	return
}

func getErrorJsonWithLineNumbers(input string, errLn int) (jsonWithLineNumbers string) {
	jsonWithLineNumbers = "\n"
	inputString := input

	n := strings.Count(inputString, "\n")
	if len(inputString) > 0 && !strings.HasSuffix(inputString, "\n") {
		n++
	}
	fmtString := fmt.Sprintf("%s%d%s", "%s%", len(strconv.Itoa(n)), "d: %s\n")

	scanner := bufio.NewScanner(strings.NewReader(string(input)))
	// Set some significant buffer to scanner (lines up to 1Mb)
	// the default would end up throwing scan errors
	// therefore the rest of the output woud be skipped
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, 16*bufio.MaxScanTokenSize)
	i := 1
	for scanner.Scan() {
		scannerText := scanner.Text()
		// Because we increased the scanner capacity the line can be too long
		// We trim it here (1Kb) for readability and add a short explanation at the end
		if len(scannerText) > 1024 {
			scannerText = scannerText[0:1024] + " ... (skipped too long output)"
		}
		if len(strings.TrimSpace(scannerText)) > 0 {
			fmtStringRow := "%s"
			if coloredError && i == errLn {
				fmtStringRow = "\033[31m%s\033[0m"
			}
			jsonWithLineNumbers = fmt.Sprintf(fmtString, jsonWithLineNumbers, i, fmt.Sprintf(fmtStringRow, scannerText))
		}
		i++
	}

	// We reached an error in the scanner, so output it
	err := scanner.Err()
	if err != nil {
		jsonWithLineNumbers = fmt.Sprintf("%s-----------\nText scanner error: %s", jsonWithLineNumbers, err.Error())
		// The manifest is just too long, add advice
		if err == bufio.ErrTooLong {
			jsonWithLineNumbers = fmt.Sprintf("%s\nSome fields are too long, consider splitting tests or reducing datasets", jsonWithLineNumbers)
		}
		jsonWithLineNumbers = fmt.Sprintf("%s\n-----------\n", jsonWithLineNumbers)
	}

	return
}

func lineAndCharacter(input string, offset int) (line int, character int, err error) {
	if offset > len(input) || offset < 0 {
		return 0, 0, fmt.Errorf("Couldn't find offset %d within the input.", offset)
	}
	// humans count line from 1
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
