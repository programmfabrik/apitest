package api

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/programmfabrik/apitest/pkg/lib/csv"
	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/programmfabrik/apitest/pkg/lib/util"
	"github.com/programmfabrik/golib"
)

type Response struct {
	StatusCode  *int
	Headers     map[string]any
	HeaderFlat  map[string]any // ":control" is an object, so we must use "any" here
	Cookies     []*http.Cookie
	Body        []byte
	BodyControl jsutil.Object
	Format      ResponseFormat

	ReqDur      time.Duration
	BodyLoadDur time.Duration
}

func httpHeaderToMap(header http.Header) (headers map[string]any, err error) {
	headers = map[string]any{}
	for k, h := range header {
		headers[k] = h
	}
	return headers, nil
}

type ResponseSerialization struct {
	StatusCode  *int                   `yaml:"statuscode,omitempty" json:"statuscode,omitempty"`
	Headers     map[string]any         `yaml:"header" json:"header,omitempty"`
	Cookies     map[string]http.Cookie `yaml:"cookie" json:"cookie,omitempty"`
	Body        any                    `yaml:"body" json:"body,omitempty"`
	BodyControl jsutil.Object          `yaml:"body:control" json:"body:control,omitempty"`
	Format      ResponseFormat         `yaml:"format" json:"format"`
}

type responseSerializationInternal struct {
	ResponseSerialization
	HeaderFlat map[string]any `json:"header_flat,omitempty"`
}

type responseFormatCSV struct {
	Comma string `json:"comma,omitempty"`
}

type responseFormatXLSX struct {
	SheetIdx int `json:"sheet_idx,omitempty"`
}

const (
	responseTypeXml    string = "xml"
	responseTypeXml2   string = "xml2"
	responseTypeHtml   string = "html"
	responseTypeXhtml  string = "xhtml"
	responseTypeXlsx   string = "xlsx"
	responseTypeCsv    string = "csv"
	responseTypeBinary string = "binary"
	responseTypeText   string = "text"
)

type ResponseFormat struct {
	IgnoreBody bool               `json:"-"`    // if true, do not try to parse the body (since it is not expected in the response)
	Type       string             `json:"type"` // default "json", allowed: "csv", "json", "xml", "xml2", "html", "xhtml", "binary", "text", "xlsx"
	CSV        responseFormatCSV  `json:"csv"`  // ignored if type != "csv"
	XLSX       responseFormatXLSX `json:"xlsx"` // ignored if type != "xlsx"
	PreProcess *preProcess        `json:"pre_process,omitempty"`
}

func NewResponse(statusCode *int,
	headersAny map[string]any,
	cookies []*http.Cookie,
	body io.Reader,
	bodyControl jsutil.Object,
	bodyFormat ResponseFormat,
) (res Response, err error) {

	headerFlat := map[string]any{}
	headers := map[string]any{}

	// parse headers and set HeaderFlat if the values are string
	for key, value := range headersAny {
		switch v := value.(type) {
		case string:
			headerFlat[key] = v
			continue
		case []any:
			headerS := []string{}
			for _, item := range v {
				switch v2 := item.(type) {
				case string:
					headerS = append(headerS, v2)
					continue
				default:
					return res, fmt.Errorf("unknown type %T in header %q", v2, key)
				}
			}
			headers[key] = headerS
			continue
		case []string:
			headers[key] = v
			continue
		case map[string]any: // check if that is a control
			if strings.HasSuffix(key, ":control") {
				headerFlat[key] = v
				continue
			}
		}
		// all valid cases continue above
		return res, fmt.Errorf("unknown type %T in header %q", value, key)
	}

	res = Response{
		StatusCode:  statusCode,
		Headers:     headers,
		BodyControl: bodyControl,
		HeaderFlat:  headerFlat,
		Cookies:     cookies,
		Format:      bodyFormat,
	}
	if body != nil {
		start := time.Now()
		res.Body, err = io.ReadAll(body)
		if err != nil {
			return res, err
		}
		res.BodyLoadDur = time.Since(start)
	}
	return res, nil
}

func NewResponseFromSpec(spec ResponseSerialization) (res Response, err error) {
	var body io.Reader
	if spec.Body != nil {
		bodyBytes, err := jsutil.Marshal(spec.Body)
		if err != nil {
			return res, err
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Build standard cookies bag from spec map
	var cookies []*http.Cookie
	if len(spec.Cookies) > 0 {
		cookies = make([]*http.Cookie, 0)
		for _, rck := range spec.Cookies {
			cookies = append(cookies, &rck)
		}
	}

	return NewResponse(spec.StatusCode, spec.Headers, cookies, body, spec.BodyControl, spec.Format)
}

// splitLines is a helper function needed for format "text"
func splitLines(s string) (lines jsutil.Array) {
	lines = jsutil.Array{}
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// ServerResponseToGenericJSON parse response from server. convert xml, csv, binary to json if necessary
func (response Response) ServerResponseToGenericJSON(responseFormat ResponseFormat, bodyOnly bool) (res any, err error) {
	var (
		bodyJSON      any
		bodyData      []byte
		resp          Response
		headerFlat    map[string]any
		headersAny    map[string]any
		responseJSON  responseSerializationInternal
		responseBytes []byte
	)

	if responseFormat.PreProcess != nil {
		resp, err = responseFormat.PreProcess.runPreProcess(response)
		if err != nil {
			return res, fmt.Errorf("could not pre process response: %w", err)
		}
		switch responseFormat.Type {
		case "":
			// ensure the resp is json
			resp.Body = ensureJson(resp.Body)
		}
	} else {
		resp = response
	}

	switch responseFormat.Type {
	case responseTypeXml, responseTypeXml2:
		bodyData, err = util.Xml2Json(resp.Body, responseFormat.Type)
		if err != nil {
			return res, fmt.Errorf("could not marshal xml to json: %w", err)
		}
	case responseTypeHtml:
		bodyData, err = util.Html2Json(resp.Body)
		if err != nil {
			return res, fmt.Errorf("could not marshal html to json: %w", err)
		}
	case responseTypeXhtml:
		bodyData, err = util.Xhtml2Json(resp.Body)
		if err != nil {
			return res, fmt.Errorf("could not marshal xhtml to json: %w", err)
		}
	case responseTypeXlsx:
		bodyData, err = util.Xlsx2Json(resp.Body, responseFormat.XLSX.SheetIdx)
		if err != nil {
			return res, fmt.Errorf("could not marshal xlsx to json: %w", err)
		}
	case responseTypeCsv:
		runeComma := ','
		if responseFormat.CSV.Comma != "" {
			runeComma = []rune(responseFormat.CSV.Comma)[0]
		}

		csvData, err := csv.GenericCSVToMap(resp.Body, runeComma)
		if err != nil {
			return res, fmt.Errorf("could not parse csv: %w", err)
		}

		bodyData, err = jsutil.Marshal(csvData)
		if err != nil {
			return res, fmt.Errorf("could not marshal csv to json: %w", err)
		}
	case responseTypeBinary:
		// We have another file format (binary). We thereby take the md5 Hash of the body and compare that one
		hasher := md5.New()
		hasher.Write([]byte(resp.Body))
		jsonObject := jsutil.Object{
			"md5sum": jsutil.String(hex.EncodeToString(hasher.Sum(nil))),
		}
		bodyData, err = jsutil.Marshal(jsonObject)
		if err != nil {
			return res, fmt.Errorf("could not marshal body with md5sum to json: %w", err)
		}
	case responseTypeText:
		// render the content as text
		bodyText := string(resp.Body)
		bodyTextTrimmed := strings.TrimSpace(bodyText)
		jsonObject := jsutil.Object{
			"text":         jsutil.String(bodyText),
			"text_trimmed": jsutil.String(bodyTextTrimmed),
			"lines":        splitLines(bodyText),
			"float64":      nil,
			"int64":        nil,
		}
		// try to parse the string as float and int
		// ignore errors silently in case the text is not numerical
		n, err2 := strconv.ParseFloat(bodyTextTrimmed, 64)
		if err2 == nil {
			jsonObject["float64"] = n
			jsonObject["int64"] = int64(n)
		}

		bodyData, err = jsutil.Marshal(jsonObject)
		if err != nil {
			return res, fmt.Errorf("could not marshal body to text (string): %w", err)
		}
	case "":
		// no specific format, we assume a json, and thereby try to unmarshal it into our body
		bodyData = resp.Body
	default:
		return res, fmt.Errorf("Invalid response format '%s'", responseFormat.Type)
	}

	headerFlat = map[string]any{}
	headersAny = map[string]any{}
	for key, value := range resp.Headers {
		headersAny[key] = value
		values := value.([]string) // this must be []string, if not this panics
		headerFlat[key] = strings.Join(values, ";")
	}

	responseJSON = responseSerializationInternal{
		ResponseSerialization: ResponseSerialization{
			StatusCode: resp.StatusCode,
			Headers:    headersAny,
		},
		HeaderFlat: headerFlat,
	}

	// Build cookies map from standard bag
	if len(resp.Cookies) > 0 {
		responseJSON.Cookies = make(map[string]http.Cookie)
		for _, ck := range resp.Cookies {
			if ck == nil {
				continue
			}
			responseJSON.Cookies[ck.Name] = *ck
		}
	}

	// if the body should not be ignored, serialize the parsed/converted body
	if !responseFormat.IgnoreBody {

		if len(bodyData) > 0 {
			err = jsutil.Unmarshal(bodyData, &bodyJSON)
			if err != nil {
				return res, err
			}
		}

		if bodyOnly {
			return bodyJSON, nil
		}

		responseJSON.Body = &bodyJSON
	}

	responseBytes, err = jsutil.Marshal(responseJSON)
	if err != nil {
		return res, err
	}

	err = jsutil.Unmarshal(responseBytes, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

// ToGenericJSON parse expected response
func (response Response) ToGenericJSON() (res any, err error) {
	var (
		bodyJSON      any
		responseJSON  responseSerializationInternal
		responseBytes []byte
	)

	// We have a json, and thereby try to unmarshal it into our body
	if len(response.Body) > 0 {
		err = jsutil.Unmarshal(response.Body, &bodyJSON)
		if err != nil {
			return res, err
		}
	}

	responseJSON = responseSerializationInternal{
		ResponseSerialization: ResponseSerialization{
			StatusCode:  response.StatusCode,
			Headers:     response.Headers,
			BodyControl: response.BodyControl,
		},
		HeaderFlat: response.HeaderFlat,
	}

	// Build cookies map from standard bag
	if len(response.Cookies) > 0 {
		responseJSON.Cookies = make(map[string]http.Cookie)
		for _, ck := range response.Cookies {
			if ck == nil {
				continue
			}
			responseJSON.Cookies[ck.Name] = *ck
		}
	}

	// necessary because check for <nil> against missing body would fail, but must succeed
	if bodyJSON != nil {
		responseJSON.Body = &bodyJSON
	}

	responseBytes, err = jsutil.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	err = jsutil.Unmarshal(responseBytes, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (response Response) ServerResponseToJsonString(bodyOnly bool) (jstring string, err error) {
	var (
		genericJSON any
		bytes       []byte
	)

	genericJSON, err = response.ServerResponseToGenericJSON(response.Format, bodyOnly)
	if err != nil {
		return "", fmt.Errorf("formatting response: %w", err)
	}

	bytes, err = golib.JsonBytesIndent(genericJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting response: %w", err)
	}

	return string(bytes), nil
}

func (response Response) ToString() (s string) {
	var (
		headersString string
		bodyString    string
		err           error
		resp          Response
		statuscode    int
	)

	for k, v := range response.Headers {
		switch v2 := v.(type) {
		case []string:
			value := ""
			for _, iv := range v2 {
				value = fmt.Sprintf("%s %s", value, iv)
			}
			if strings.TrimSpace(value) == "" {
				continue
			}
			headersString = fmt.Sprintf("%s\n%s:%s", headersString, k, value)
		}
	}

	resp = response

	// for logging, always show the body
	resp.Format.IgnoreBody = false
	switch resp.Format.Type {
	case responseTypeXml,
		responseTypeXml2,
		responseTypeHtml,
		responseTypeXhtml,
		responseTypeXlsx,
		responseTypeCsv,
		responseTypeBinary,
		responseTypeText:
		bodyString, err = resp.ServerResponseToJsonString(true)
		if err != nil {
			bodyString = "[BINARY DATA NOT DISPLAYED]\n\n"
		}
	default:
		if utf8.Valid(resp.Body) {
			bodyString = string(resp.Body)
		} else {
			bodyString = "[BINARY DATA NOT DISPLAYED]\n\n"
		}
	}

	if resp.StatusCode != nil {
		statuscode = *resp.StatusCode
	}
	return fmt.Sprintf("%d\n%s\n\n%s", statuscode, headersString, bodyString)
}
