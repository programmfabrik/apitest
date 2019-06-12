package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"github.com/programmfabrik/fylr-apitest/lib/util"
	"io"
	"io/ioutil"
)

type Response struct {
	statusCode int
	headers    map[string][]string
	body       []byte
}

type ResponseSerialization struct {
	StatusCode int                 `yaml:"statuscode" json:"statuscode"`
	Headers    map[string][]string `yaml:"header" json:"header,omitempty"`
	Body       util.GenericJson    `yaml:"body" json:"body,omitempty"`
}

func NewResponse(statusCode int, headers map[string][]string, body io.Reader) (res Response, err error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return res, err
	}
	return Response{statusCode: statusCode, headers: headers, body: bodyBytes}, nil
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
		return NewResponse(spec.StatusCode, spec.Headers, bytes.NewReader(bodyBytes))
	} else {
		return NewResponse(spec.StatusCode, nil, bytes.NewReader(bodyBytes))
	}
}

func (response Response) ToGenericJson() (res util.GenericJson, err error) {
	var gj util.GenericJson
	bodyBytes := response.Body()

	//Check mimetype of body
	bodyMimeType, _ := mimetype.Detect(bodyBytes)
	if bodyMimeType == "text/plain" || bodyMimeType == "application/json" {
		// We have a json, and thereby try to unmarshal it into our body
		if err = cjson.Unmarshal(bodyBytes, &gj); err != nil {
			return res, err
		}

		responseJSON := ResponseSerialization{
			StatusCode: response.statusCode,
		}
		if len(response.headers) > 0 {
			responseJSON.Headers = response.headers
		}
		if gj != nil {
			responseJSON.Body = &gj
		}

		responseBytes, err := cjson.Marshal(responseJSON)
		if err != nil {
			return res, err
		}
		cjson.Unmarshal(responseBytes, &res)
		return res, nil
	} else {
		// We have another file format (binary). We thereby take the md5 Hash of the body and compare that one
		hasher := md5.New()
		hasher.Write([]byte(bodyBytes))
		jsonObject := util.JsonObject{
			"statuscode": util.JsonNumber(response.statusCode),
			"body": util.JsonObject{
				"BinaryFileHash": util.JsonString(hex.EncodeToString(hasher.Sum(nil))),
			},
		}
		return jsonObject, nil
	}
}

func (response Response) ToJsonString() (string, error) {
	gj, err := response.ToGenericJson()
	if err != nil {
		return "", fmt.Errorf("error formatting respsone: %s", err)
	}
	json, err := cjson.Marshal(gj)
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	return string(json), nil
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
		headersString = fmt.Sprintf("%s\n%s:%s", headersString, k, value)
	}

	bodyMimeType, _ := mimetype.Detect(response.Body())
	if bodyMimeType == "text/plain" || bodyMimeType == "application/json" {
		return fmt.Sprintf("%d\n%s\n\n%s", response.statusCode, headersString, string(response.Body()))

	} else {
		return fmt.Sprintf("%d\n%s\n\n%s", response.statusCode, headersString, "[BINARY DATA NOT DISPLAYED]")

	}
}
