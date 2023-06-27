package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

	jsonStr, err := mv.JsonIndent("", " ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not convert to json")
	}
	return jsonStr, nil
}

// Xhtml2Json parses the raw xhtml data and converts it into a json string
func Xhtml2Json(rawXhtml []byte) ([]byte, error) {
	var (
		mv  mxj.Map
		err error
	)

	mv, err = mxj.NewMapXml(rawXhtml)
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not parse xhtml")
	}

	jsonStr, err := mv.JsonIndent("", " ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not convert to json")
	}
	return jsonStr, nil
}

// Html2Json parses the raw html data and converts it into a json string
func Html2Json(rawHtml []byte) ([]byte, error) {
	var (
		htmlDoc *goquery.Document
		err     error
	)

	htmlDoc, err = goquery.NewDocumentFromReader(bytes.NewReader(rawHtml))
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not parse html")
	}

	htmlData := parseHtmlNode(htmlDoc.Selection)
	jsonStr, err := json.MarshalIndent(htmlData, "", " ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not convert html to json")
	}

	return jsonStr, nil
}

func parseHtmlNode(node *goquery.Selection) map[string]interface{} {
	tagName := node.Get(0).Data
	tagData := map[string]interface{}{}

	// include attributes
	attrs := map[string]interface{}{}
	for _, attr := range node.Get(0).Attr {
		attrs[attr.Key] = attr.Val
	}
	if len(attrs) > 0 {
		tagData["#attr"] = attrs
	}

	// recursively parse child nodes
	childrenByName := map[string][]interface{}{}
	node.Children().Each(func(i int, childNode *goquery.Selection) {
		for childName, childContent := range parseHtmlNode(childNode) {
			childrenByName[childName] = append(childrenByName[childName], childContent)
		}
	})
	for childName, children := range childrenByName {
		tagData[childName] = children
	}

	text := strings.Trim(node.Text(), " \n\t")
	if len(text) > 0 && len(childrenByName) < 1 {
		// include the text only if there are no children, since goquery would render all children into a single string
		tagData["#text"] = text
	}

	// there might be an empty top level tag (eg '<!DOCTYPE')
	if tagName == "" {
		return tagData
	}

	return map[string]interface{}{
		tagName: tagData,
	}
}
