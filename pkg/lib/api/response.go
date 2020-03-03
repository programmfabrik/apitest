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

	"github.com/clbanning/mxj"
	"github.com/gabriel-vasile/mimetype"
	"github.com/k0kubun/pp"
	"github.com/pkg/errors"

	"github.com/programmfabrik/apitest/pkg/lib/cjson"
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
	Body        util.GenericJson    `yaml:"body" json:"body,omitempty"`
	BodyControl util.JsonObject     `yaml:"body:control" json:"body:control,omitempty"`
	Format      ResponseFormat      `yaml:"format" json:"format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"` // default "json", allowed: "csv", "json", "xml", "binary"
	CSV  struct {
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
	bodyBytes, err := cjson.Marshal(spec.Body)
	if err != nil {
		return res, err
	}
	// if statuscode is not explicitly set; we assume 200
	if spec.StatusCode == 0 {
		spec.StatusCode = 200
	}

	if spec.Headers != nil {
		return NewResponse(spec.StatusCode, spec.Headers, bytes.NewReader(bodyBytes), spec.BodyControl, spec.Format)
	} else {
		return NewResponse(spec.StatusCode, nil, bytes.NewReader(bodyBytes), spec.BodyControl, spec.Format)
	}
}

// ServerResponseToGenericJSON parse response from server. convert xml, csv, binary to json if necessary
func (response Response) ServerResponseToGenericJSON(responseFormat ResponseFormat) (util.GenericJson, error) {
	var (
		gj, res util.GenericJson
		err     error
	)

	pp.Println("*************************")
	pp.Println("ServerResponseToGenericJSON", responseFormat.Type)

	switch responseFormat.Type {
	case "xml":
		xmlDeclarationRegex := regexp.MustCompile(`<\?xml.*?\?>`)
		replacedXML := xmlDeclarationRegex.ReplaceAll(response.Body(), []byte{})

		mv, err := mxj.NewMapXmlSeq(replacedXML)
		if err != nil {
			return res, errors.Wrap(err, "Could not parse xml")
		}

		responseJSON, err := mv.JsonIndent("", " ")
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal xml to json")
		}

		pp.Println("responseJSON from XML:", responseJSON)
		return responseJSON, nil
	case "csv":
		runeComma := ','
		if response.Format.CSV.Comma != "" {
			runeComma = []rune(response.Format.CSV.Comma)[0]
		}

		d, err := csv.GenericCSVToMap(response.Body(), runeComma)
		if err != nil {
			return res, errors.Wrap(err, "Could not parse csv")
		}

		responseJSON, err := json.Marshal(d)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal csv to json")
		}

		pp.Println("responseJSON from CSV:", responseJSON)
		return responseJSON, nil
	case "binary":
		// We have another file format (binary). We thereby take the md5 Hash of the body and compare that one
		hasher := md5.New()
		hasher.Write([]byte(response.Body()))
		jsonObject := util.JsonObject{
			"statuscode": util.JsonNumber(response.statusCode),
			"body": util.JsonObject{
				"md5sum": util.JsonString(hex.EncodeToString(hasher.Sum(nil))),
			},
		}
		pp.Println("responseJSON from binary:", jsonObject)
		return jsonObject, nil
	default:
		// We have a json, and thereby try to unmarshal it into our body
		if err = json.Unmarshal(response.Body(), &gj); err != nil {
			return res, err
		}

		responseJSON := ResponseSerialization{
			StatusCode: response.statusCode,
			Body:       &gj,
		}

		if len(response.headers) > 0 {
			responseJSON.Headers = response.headers
		}

		responseBytes, err := cjson.Marshal(responseJSON)
		if err != nil {
			return res, err
		}
		cjson.Unmarshal(responseBytes, &res)

		pp.Println("responseJSON for default format:", res)
		return res, nil
	}
}

// ToGenericJSON parse expected response
func (response Response) ToGenericJSON() (util.GenericJson, error) {
	var (
		gj, res util.GenericJson
		err     error
	)

	pp.Println("*************************")
	pp.Println("ToGenericJSON")

	// We have a json, and thereby try to unmarshal it into our body
	if err = cjson.Unmarshal(response.Body(), &gj); err != nil {
		return res, err
	}

	responseJSON := ResponseSerialization{
		StatusCode:  response.statusCode,
		BodyControl: response.bodyControl,
	}
	if len(response.headers) > 0 {
		responseJSON.Headers = response.headers
	}
	responseJSON.Body = &gj

	responseBytes, err := cjson.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	cjson.Unmarshal(responseBytes, &res)

	pp.Println("responseJSON:", res)
	return res, nil
}

func (response Response) ServerResponseToJSONString() (string, error) {
	gj, err := response.ServerResponseToGenericJSON(response.Format)
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	bytes, err := cjson.Marshal(gj)
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	return string(bytes), nil
}

func (response Response) Body() []byte {
	//some endpoints return empty strings;
	//since that is no valid json so we interpret it as the json null literal to
	//establish the invariant that api endpoints return json responses
	if bytes.Compare(response.body, []byte("")) == 0 {
		return []byte("null")
	}
	return response.body
}

func (response *Response) marshalBodyInto(target interface{}) (err error) {
	bodyBytes := response.Body()
	if err = cjson.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("error unmarshaling response: %s", err)
	}
	return nil
}

func (response Response) ToString() (res string) {
	headersString := ""
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

	bodyMimeType, _ := mimetype.Detect(response.Body())
	if bodyMimeType == "text/plain" || bodyMimeType == "application/json" {
		return fmt.Sprintf("%d\n%s\n\n%s", response.statusCode, headersString, string(response.Body()))

	} else {
		return fmt.Sprintf("%d\n%s\n\n%s", response.statusCode, headersString, "[BINARY DATA NOT DISPLAYED]")

	}
}
