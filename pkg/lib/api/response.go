package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/clbanning/mxj"
	"github.com/gabriel-vasile/mimetype"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/programmfabrik/apitest/pkg/lib/csv"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

type Response struct {
	statusCode  int
	headers     map[string][]string
	body        []byte
	bodyControl util.JsonObject
	Format      ResponseFormat
}

type ResponseSerialization struct {
	StatusCode  int                 `yaml:"statuscode" json:"statuscode"`
	Headers     map[string][]string `yaml:"header" json:"header,omitempty"`
	Body        interface{}         `yaml:"body" json:"body,omitempty"`
	BodyControl util.JsonObject     `yaml:"body:control" json:"body:control,omitempty"`
	Format      ResponseFormat      `yaml:"format" json:"format,omitempty"`
}

type ResponseFormat struct {
	IgnoreBody bool   `json:"-"`    // if true, do not try to parse the body (since it is not expected in the response)
	Type       string `json:"type"` // default "json", allowed: "csv", "json", "xml", "binary"
	CSV        struct {
		Comma string `json:"comma,omitempty"`
	} `json:"csv,omitempty"`
}

func NewResponse(statusCode int, headers map[string][]string, body io.Reader, bodyControl util.JsonObject, bodyFormat ResponseFormat) (res Response, err error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return res, err
	}
	return Response{statusCode: statusCode, headers: headers, body: bodyBytes, bodyControl: bodyControl, Format: bodyFormat}, nil
}

func NewResponseFromSpec(spec ResponseSerialization) (res Response, err error) {
	bodyBytes, err := json.Marshal(spec.Body)
	if err != nil {
		return res, err
	}
	// if statuscode is not explicitly set; we assume 200
	if spec.StatusCode == 0 {
		spec.StatusCode = 200
	}

	return NewResponse(spec.StatusCode, spec.Headers, bytes.NewReader(bodyBytes), spec.BodyControl, spec.Format)
}

// ServerResponseToGenericJSON parse response from server. convert xml, csv, binary to json if necessary
func (response Response) ServerResponseToGenericJSON(responseFormat ResponseFormat, bodyOnly bool) (interface{}, error) {
	var (
		res, bodyJSON interface{}
		bodyData      []byte
		err           error
	)

	switch responseFormat.Type {
	case "xml":
		xmlDeclarationRegex := regexp.MustCompile(`<\?xml.*?\?>`)
		replacedXML := xmlDeclarationRegex.ReplaceAll(response.Body(), []byte{})

		mv, err := mxj.NewMapXmlSeq(replacedXML)
		if err != nil {
			return res, errors.Wrap(err, "Could not parse xml")
		}

		bodyData, err = mv.JsonIndent("", " ")
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal xml to json")
		}
	case "csv":
		runeComma := ','
		if responseFormat.CSV.Comma != "" {
			runeComma = []rune(responseFormat.CSV.Comma)[0]
		}

		csvData, err := csv.GenericCSVToMap(response.Body(), runeComma)
		if err != nil {
			return res, errors.Wrap(err, "Could not parse csv")
		}

		bodyData, err = json.Marshal(csvData)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal csv to json")
		}
	case "binary":
		// We have another file format (binary). We thereby take the md5 Hash of the body and compare that one
		hasher := md5.New()
		hasher.Write([]byte(response.Body()))
		JsonObject := util.JsonObject{
			"md5sum": util.JsonString(hex.EncodeToString(hasher.Sum(nil))),
		}
		bodyData, err = json.Marshal(JsonObject)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal body with md5sum to json")
		}
	default:
		// We assume a json, and thereby try to unmarshal it into our body
		bodyData = response.Body()
	}

	responseJSON := ResponseSerialization{
		StatusCode: response.statusCode,
	}
	if len(response.headers) > 0 {
		responseJSON.Headers = response.headers
	}

	// if the body should not be ignored, serialize the parsed/converted body
	if !responseFormat.IgnoreBody {
		err = json.Unmarshal(bodyData, &bodyJSON)
		if err != nil {
			return res, err
		}

		if bodyOnly {
			return bodyJSON, nil
		}

		responseJSON.Body = &bodyJSON
	}

	responseBytes, err := json.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	json.Unmarshal(responseBytes, &res)

	return res, nil
}

// ToGenericJSON parse expected response
func (response Response) ToGenericJSON() (interface{}, error) {
	var (
		bodyJSON, res interface{}
		err           error
	)

	// We have a json, and thereby try to unmarshal it into our body
	err = json.Unmarshal(response.Body(), &bodyJSON)
	if err != nil {
		return res, err
	}

	responseJSON := ResponseSerialization{
		StatusCode:  response.statusCode,
		BodyControl: response.bodyControl,
	}
	if len(response.headers) > 0 {
		responseJSON.Headers = response.headers
	}

	// necessary because check for <nil> against missing body would fail, but must succeed
	if bodyJSON != nil {
		responseJSON.Body = &bodyJSON
	}

	responseBytes, err := json.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	json.Unmarshal(responseBytes, &res)

	return res, nil
}

func (response Response) ServerResponseToJsonString(bodyOnly bool) (string, error) {
	genericJSON, err := response.ServerResponseToGenericJSON(response.Format, bodyOnly)
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	bytes, err := json.MarshalIndent(genericJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	return string(bytes), nil
}

func (response Response) Body() []byte {
	// some endpoints return empty strings;
	// since that is no valid json so we interpret it as the json null literal to
	// establish the invariant that api endpoints return json responses
	if bytes.Compare(response.body, []byte("")) == 0 {
		return []byte("null")
	}
	return response.body
}

func (response *Response) marshalBodyInto(target interface{}) (err error) {
	bodyBytes := response.Body()
	if err = json.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("error unmarshaling response: %s", err)
	}
	return nil
}

func (response Response) ToString() string {
	var (
		headersString string
		bodyString    string
		err           error
	)

	for k, v := range response.headers {
		value := ""
		for _, iv := range v {
			value = fmt.Sprintf("%s %s", value, iv)
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		headersString = fmt.Sprintf("%s\n%s:%s", headersString, k, value)
	}

	// try to determine the mime type from the body
	bodyMimeType, _ := mimetype.Detect(response.Body())
	switch bodyMimeType {
	case "text/plain":
		bodyString = string(response.Body())
	case "application/json":
		// for logging, always show the body
		response.Format.IgnoreBody = false
		bodyString, err = response.ServerResponseToJsonString(true)
		if err != nil {
			logrus.Warnf("could not parse JSON response body: %s", err)
			bodyString = string(response.Body())
		}
	default:
		if utf8.Valid(response.Body()) {
			bodyString = string(response.Body())
		} else {
			bodyString = fmt.Sprintf("[BINARY DATA NOT DISPLAYED]\n\n")
		}
	}

	return fmt.Sprintf("%d\n%s\n\n%s", response.statusCode, headersString, bodyString)
}
