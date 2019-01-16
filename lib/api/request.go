package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/programmfabrik/fylr-apitest/lib/util"
)

type Request struct {
	endpoint    string
	method      string
	queryParams map[string]string
	headers     map[string]string
	body        util.GenericJson
	bodyType    string

	buildPolicy func(Request) (additionalHeaders map[string]string, body io.Reader, err error)

	// bool store or not
	DoStore bool

	manifestDir string
}

type RequestSerialization struct {
	Endpoint    string                 `yaml:"endpoint" json:"endpoint"`
	Method      string                 `yaml:"method" json:"method"`
	QueryParams map[string]interface{} `yaml:"query_params" json:"query_params"`
	Headers     map[string]string      `yaml:"header" json:"header"`
	BodyType    string                 `yaml:"body_type" json:"body_type"`
	Body        util.GenericJson       `yaml:"body" json:"body"`
}

func NewRequest(requestSerialization RequestSerialization, manifestDir string) (res Request) {
	res.setBuildPolicy(requestSerialization.BodyType)

	res.DoStore = true
	res.endpoint = requestSerialization.Endpoint
	res.method = requestSerialization.Method
	res.headers = requestSerialization.Headers
	res.body = requestSerialization.Body
	res.manifestDir = manifestDir
	res.bodyType = requestSerialization.BodyType

	//Convert queryParams to string and add them
	res.queryParams = make(map[string]string, 0)
	for key, val := range requestSerialization.QueryParams {
		switch t := val.(type) {
		case string:
			res.queryParams[key] = t
		case float64:
			res.queryParams[key] = strconv.FormatFloat(t, 'f', -1, 64)
		case int:
			res.queryParams[key] = fmt.Sprintf("%d", t)
		default:
			jsonVal, _ := json.Marshal(t)
			//TODO: Errorhandling if json.Marshal errors
			res.queryParams[key] = string(jsonVal)
		}
	}

	return res
}

func (request *Request) setBuildPolicy(bodyType string) {
	switch bodyType {
	case "multipart":
		request.buildPolicy = buildMultipart
	case "urlencoded":
		request.buildPolicy = buildUrlencoded
	default:
		request.buildPolicy = buildRegular
	}
}

func (request Request) buildHttpRequest(serverUrl string, token string) (res *http.Request, err error) {
	requestUrl := fmt.Sprintf("%s/%s", serverUrl, request.endpoint)
	additionalHeaders, body, err := request.buildPolicy(request)
	if err != nil {
		return res, fmt.Errorf("error executing buildpolicy: %s", err)
	}
	res, err = http.NewRequest(request.method, requestUrl, body)
	if err != nil {
		return res, fmt.Errorf("error creating new request")
	}

	q := res.URL.Query()
	for key, val := range request.queryParams {
		q.Add(key, val)
	}
	res.URL.RawQuery = q.Encode()

	for key, val := range request.headers {
		res.Header.Add(key, val)
	}

	additionalHeaders["x-easydb-token"] = token
	for key, val := range additionalHeaders {
		res.Header.Add(key, val)
	}
	return res, nil
}

func (request Request) ToString(session Session) (res string) {
	httpRequest, err := request.buildHttpRequest(session.serverUrl, session.token)
	if err != nil {
		return fmt.Sprintf("could not build httpRequest: %s", err)
	}

	var dumpBody bool
	if request.bodyType == "multipart" {
		dumpBody = false
	} else {
		dumpBody = true
	}
	resBytes, err := httputil.DumpRequest(httpRequest, dumpBody)
	if err != nil {
		return fmt.Sprintf("could not dump httpRequest: %s", err)
	}
	return string(resBytes)
}
