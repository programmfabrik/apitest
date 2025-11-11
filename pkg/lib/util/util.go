package util

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/clbanning/mxj"
	libcsv "github.com/programmfabrik/apitest/pkg/lib/csv"
	"github.com/programmfabrik/golib"
	"github.com/xuri/excelize/v2"
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

func GetStringFromInterface(queryParam any) (v string, err error) {
	switch t := queryParam.(type) {
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case int:
		return fmt.Sprintf("%d", t), nil
	default:
		var jsonVal []byte
		jsonVal, err = json.Marshal(t)
		return string(jsonVal), err
	}
}

// Xml2Json parses the raw xml data and converts it into a json string
// there are 2 formats for the result json:
// - "xml": use mxj.NewMapXmlSeq (more complex format including #seq)
// - "xml2": use mxj.NewMapXmlSeq (simpler format)
func Xml2Json(rawXml []byte, format string) (jsonStr []byte, err error) {
	var (
		mv                  mxj.Map
		xmlDeclarationRegex *regexp.Regexp
		replacedXML         []byte
	)

	xmlDeclarationRegex = regexp.MustCompile(`<\?xml.*?\?>`)
	replacedXML = xmlDeclarationRegex.ReplaceAll(rawXml, []byte{})

	switch format {
	case "xml":
		mv, err = mxj.NewMapXmlSeq(replacedXML)
	case "xml2":
		mv, err = mxj.NewMapXml(replacedXML)
	default:
		return []byte{}, fmt.Errorf("Unknown format %s", format)
	}

	if err != nil {
		return []byte{}, fmt.Errorf("could not parse xml: %w", err)
	}

	jsonStr, err = mv.JsonIndent("", " ")
	if err != nil {
		return []byte{}, fmt.Errorf("could not convert to json: %w", err)
	}

	return jsonStr, nil
}

// Xhtml2Json parses the raw xhtml data and converts it into a json string
func Xhtml2Json(rawXhtml []byte) (jsonStr []byte, err error) {
	var (
		mv mxj.Map
	)

	mv, err = mxj.NewMapXml(rawXhtml)
	if err != nil {
		return []byte{}, fmt.Errorf("could not parse xhtml: %w", err)
	}

	jsonStr, err = mv.JsonIndent("", " ")
	if err != nil {
		return []byte{}, fmt.Errorf("could not convert to json: %w", err)
	}

	return jsonStr, nil
}

// Xlsx2Json parses the raw xlsx data and converts it into a json string.
// Only the content is parsed, all formatting etc is discarded.
// The result structure is the same as for CSV.
func Xlsx2Json(rawXlsx []byte, sheetIdx int) (jsonStr []byte, err error) {
	var (
		csvBuf bytes.Buffer
		xlsx   *excelize.File
	)

	// parse xlsx raw data
	xlsx, err = excelize.OpenReader(bytes.NewReader(rawXlsx))
	if err != nil {
		return []byte{}, fmt.Errorf("could not read raw xlsx data: %w", err)
	}
	defer xlsx.Close()

	// check if the requested sheet idx is valid
	if sheetIdx < 0 || sheetIdx >= xlsx.SheetCount {
		return []byte{}, fmt.Errorf("could not read xlsx sheet: idx %d invalid: expect idx between 0 and %d", sheetIdx, xlsx.SheetCount-1)
	}

	xlsxSheet := xlsx.GetSheetName(sheetIdx)
	if xlsxSheet == "" {
		return []byte{}, fmt.Errorf("could not parse xlsx: idx %d invalid: no sheets found", sheetIdx)
	}

	// read xlsx xlsxRows
	xlsxRows, err := xlsx.GetRows(xlsxSheet)
	if err != nil {
		return []byte{}, fmt.Errorf("could not parse xlsx: %w", err)
	}

	// built dummy csv to convert it into json
	csvWriter := csv.NewWriter(&csvBuf)
	csvWriter.Comma = ','
	for _, xlsxRow := range xlsxRows {
		err = csvWriter.Write(xlsxRow)
		if err != nil {
			return []byte{}, fmt.Errorf("could not convert xlsx into csv: %w", err)
		}
	}
	csvWriter.Flush()

	// parse dummy csv to convert it into json
	csvData, err := libcsv.GenericCSVToMap(csvBuf.Bytes(), csvWriter.Comma)
	if err != nil {
		return []byte{}, fmt.Errorf("could not parse csv: %w", err)
	}

	jsonStr, err = json.Marshal(csvData)
	if err != nil {
		return []byte{}, fmt.Errorf("could not convert to json: %w", err)
	}

	return jsonStr, nil
}

// Html2Json parses the raw html data and converts it into a json string
func Html2Json(rawHtml []byte) (jsonStr []byte, err error) {
	var (
		htmlDoc  *goquery.Document
		htmlData map[string]any
	)

	htmlDoc, err = goquery.NewDocumentFromReader(bytes.NewReader(rawHtml))
	if err != nil {
		return []byte{}, fmt.Errorf("could not parse html: %w", err)
	}

	htmlData = map[string]any{}
	htmlDoc.Selection.Contents().Each(func(_ int, node *goquery.Selection) {
		switch node.Get(0).Type {
		case html.ElementNode:
			htmlData = parseHtmlNode(node)
			return
		default:
			return
		}
	})

	jsonStr, err = golib.JsonBytesIndent(htmlData, "", " ")
	if err != nil {
		return []byte{}, fmt.Errorf("could not convert html to json: %w", err)
	}

	return jsonStr, nil
}

// parseHtmlNode recursivly parses the html node and adds it to a map
// the resulting structure is the same as the result of format "xml2" (using mxj.NewMapXmlSeq)
func parseHtmlNode(node *goquery.Selection) (htmlMap map[string]any) {
	var (
		tagData        map[string]any
		childrenByName map[string][]any
		comments       []string
	)

	childrenByName = map[string][]any{}
	comments = []string{}
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

	tagData = map[string]any{}

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
