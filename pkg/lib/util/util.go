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
	"github.com/programmfabrik/golib"
	"golang.org/x/net/html"
)

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func RemoveFromJsonArray(input []any, removeIndex int) (output []any) {
	output = make([]any, len(input))
	copy(output, input)

	// Remove the element at index i from a.
	copy(output[removeIndex:], input[removeIndex+1:]) // Shift a[i+1:] left one index.
	output[len(output)-1] = nil                       // Erase last element (write zero value).
	output = output[:len(output)-1]                   // Truncate slice.

	return output
}

func GetStringFromInterface(queryParam any) (string, error) {
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

	htmlData := map[string]any{}
	htmlDoc.Selection.Contents().Each(func(_ int, node *goquery.Selection) {
		switch node.Get(0).Type {
		case html.ElementNode:
			htmlData = parseHtmlNode(node)
			return
		default:
			return
		}
	})

	jsonStr, err := golib.JsonBytesIndent(htmlData, "", " ")
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not convert html to json")
	}

	return jsonStr, nil
}

// parseHtmlNode recursivly parses the html node and adds it to a map
// the resulting structure is the same as the result of format "xml2" (using mxj.NewMapXmlSeq)
func parseHtmlNode(node *goquery.Selection) map[string]any {
	tagData := map[string]any{}

	childrenByName := map[string][]any{}
	comments := []string{}

	node.Contents().Each(func(i int, content *goquery.Selection) {
		switch content.Get(0).Type {
		case html.ElementNode:
			// recursively parse child nodes
			for childName, childContent := range parseHtmlNode(content) {
				childrenByName[childName] = append(childrenByName[childName], childContent)
			}
		case html.CommentNode:
			comments = append(comments, strings.Trim(content.Get(0).Data, " \n\t"))
		default:
			return
		}
	})

	// include attributes
	for _, attr := range node.Get(0).Attr {
		tagData["-"+attr.Key] = attr.Val
	}

	// include comments
	if len(comments) == 1 {
		tagData["#comment"] = comments[0]
	} else if len(comments) > 1 {
		tagData["#comment"] = comments
	}

	// include children
	for childName, children := range childrenByName {
		if len(children) == 0 {
			continue
		}
		if len(children) == 1 {
			tagData[childName] = children[0]
			continue
		}
		tagData[childName] = children
	}

	// include tag text only if there are no children, since goquery would render all children into a single string
	if len(childrenByName) == 0 {
		text := strings.Trim(node.Text(), " \n\t")
		if len(text) > 0 {
			tagData["#text"] = text
		}
	}

	return map[string]any{
		node.Get(0).Data: tagData,
	}
}
