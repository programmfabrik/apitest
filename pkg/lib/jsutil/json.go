package jsutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
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
	coloredError bool
)

func init() {
	coloredError = true
}

// NumberEqual is comparing the string representation of the json.Number.
// It fails to compare different formats, 1e10 != 10000000000, although it is the same mathematical value.
func NumberEqual(numberExp, numberGot Number) (eq bool) {
	return numberExp == numberGot
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
	// Remove //, /* comments plus tailing commas
	return UnmarshalPlain(jsonc.ToJSON(input), output)
}

// unmarshalPlainOpts configures the json/v2 engine to match the semantics of
// the former v1 decoder exactly: v1 defaults (case-insensitive field names,
// duplicate keys last-wins, invalid UTF-8 replaced) plus DisallowUnknownFields
// and, via anyUseNumberUnmarshaler, UseNumber for every `any` destination.
var unmarshalPlainOpts = jsonv2.JoinOptions(
	json.DefaultOptionsV1(),
	jsonv2.RejectUnknownMembers(true),
	jsonv2.WithUnmarshalers(jsonv2.UnmarshalFromFunc(anyUseNumberUnmarshaler)),
)

// UnmarshalPlain decodes the input bytes into the output like Unmarshal, but
// without the cjson comment / trailing comma handling. Use it for JSON which
// was produced by marshaling or received from a server, that JSON cannot
// contain comments and running the comment passes over it is wasted work.
//
// It runs on the json/v2 engine: unlike the v1 json.Decoder, which copies the
// input into its internal buffer and decodes generic values through reflect,
// jsontext parses the []byte in place (bytes.Buffer fast path) and the `any`
// tree is built by a plain token loop. Semantics are kept at v1: see
// unmarshalPlainOpts, and like the old Decoder.Decode, only the first JSON
// value is read, trailing data is ignored. On error the old v1 decode is
// re-run on the same input so error values and texts stay byte-identical.
func UnmarshalPlain(input []byte, output any) (err error) {
	dec := jsontext.NewDecoder(bytes.NewBuffer(input), unmarshalPlainOpts)
	err = jsonv2.UnmarshalDecode(dec, output, unmarshalPlainOpts)
	if err != nil {
		v1Err := unmarshalPlainV1(input, output)
		if v1Err == nil {
			// v2 was stricter than v1 here; the v1 decode filled output.
			return nil
		}
		return getIndepthJsonError(input, v1Err)
	}
	return nil
}

// unmarshalPlainV1 is the pre-json/v2 implementation of UnmarshalPlain. It is
// kept as the error path of UnmarshalPlain: decoding is deterministic, so
// re-running it on failing input reproduces the exact v1 error (types
// *json.SyntaxError / *json.UnmarshalTypeError and their texts, which
// getIndepthJsonError and the tests rely on). Should the v2 path ever be
// stricter than v1 on some input, this also silently restores the v1 result.
func unmarshalPlainV1(input []byte, output any) (err error) {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	dec.UseNumber()

	// unmarshal into object
	return dec.Decode(output)
}

// anyUseNumberUnmarshaler decodes a JSON value into an `any` destination the
// way the v1 decoder with UseNumber did: Object / Array / String / Number
// (json.Number) / Bool / nil, duplicate object keys last-wins.
func anyUseNumberUnmarshaler(dec *jsontext.Decoder, out *any) error {
	v, err := decodeAnyValue(dec)
	if err != nil {
		return err
	}
	*out = v
	return nil
}

func decodeAnyValue(dec *jsontext.Decoder) (any, error) {
	switch dec.PeekKind() {
	case '{':
		if _, err := dec.ReadToken(); err != nil { // '{'
			return nil, err
		}
		obj := Object{}
		for dec.PeekKind() != '}' {
			keyTok, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			key := keyTok.String()
			val, err := decodeAnyValue(dec)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
		if _, err := dec.ReadToken(); err != nil { // '}'
			return nil, err
		}
		return obj, nil
	case '[':
		if _, err := dec.ReadToken(); err != nil { // '['
			return nil, err
		}
		arr := Array{}
		for dec.PeekKind() != ']' {
			val, err := decodeAnyValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		if _, err := dec.ReadToken(); err != nil { // ']'
			return nil, err
		}
		return arr, nil
	case '"':
		tok, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		return tok.String(), nil
	case '0':
		val, err := dec.ReadValue()
		if err != nil {
			return nil, err
		}
		return Number(val), nil
	default:
		tok, err := dec.ReadToken() // true / false / null, or a syntax error
		if err != nil {
			return nil, err
		}
		switch tok.Kind() {
		case 't':
			return true, nil
		case 'f':
			return false, nil
		case 'n':
			return nil, nil
		}
		return nil, fmt.Errorf("jsutil: unexpected token %v", tok)
	}
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
	inputString := input

	n := strings.Count(inputString, "\n")
	if len(inputString) > 0 && !strings.HasSuffix(inputString, "\n") {
		n++
	}
	fmtString := fmt.Sprintf("%s%d%s", "%", len(strconv.Itoa(n)), "d: %s")

	scanner := bufio.NewScanner(strings.NewReader(string(input)))
	// Set some significant buffer to scanner (lines up to 1Mb)
	// the default would end up throwing scan errors
	// therefore the rest of the output woud be skipped
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, 16*bufio.MaxScanTokenSize)
	i := 1

	lines := []string{}

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
			lines = append(lines, fmt.Sprintf(fmtString, i, fmt.Sprintf(fmtStringRow, scannerText)))
		}
		i++
	}
	jsonWithLineNumbers = "\n" + strings.Join(lines, "\n") + "\n"

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
