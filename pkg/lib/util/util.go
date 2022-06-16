package util

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/clbanning/mxj"
	"github.com/pkg/errors"
)

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func RemoveFromJsonArray(input []interface{}, removeIndex int) (output []interface{}) {
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

// Xml2Json parses the raw xml data and converts it into a json string
// there are 2 formats for the result json:
// - "xml": use mxj.NewMapXmlSeq (more complex format including #seq)
// - "xml2": use mxj.NewMapXmlSeq (simpler format)
func Xml2Json(rawXml []byte, format string) ([]byte, error) {
	var (
		mv  mxj.Map
		err error
	)

	xmlDeclarationRegex := regexp.MustCompile(`<\?xml.*?\?>`)
	replacedXML := xmlDeclarationRegex.ReplaceAll(rawXml, []byte{})

	switch format {
	case "xml":
		mv, err = mxj.NewMapXmlSeq(replacedXML)
	case "xml2":
		mv, err = mxj.NewMapXml(replacedXML)
	default:
		return []byte{}, errors.Errorf("Unknown format %s", format)
	}

	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not parse xml")
	}

	json, err := mv.JsonIndent("", " ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not convert to json")
	}
	return json, nil
}
