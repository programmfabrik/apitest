package api

import (
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/util"
	"io"
	"net/http"
	"net/http/httputil"
)

type Request struct {
	Endpoint    string                 `yaml:"endpoint" json:"endpoint"`
	Method      string                 `yaml:"method" json:"method"`
	QueryParams map[string]interface{} `yaml:"query_params" json:"query_params"`
	Headers     map[string]string      `yaml:"header" json:"header"`
	BodyType    string                 `yaml:"body_type" json:"body_type"`
	Body        util.GenericJson       `yaml:"body" json:"body"`

	buildPolicy func(Request) (additionalHeaders map[string]string, body io.Reader, err error)
	DoNotStore  bool
	ManifestDir string
}

func (request Request) buildHttpRequest(serverUrl string, token string) (res *http.Request, err error) {
	if request.buildPolicy == nil {
		//Set Build policy
		switch request.BodyType {
		case "multipart":
			request.buildPolicy = buildMultipart
		case "urlencoded":
			request.buildPolicy = buildUrlencoded
		default:
			request.buildPolicy = buildRegular
		}
	}
	//Render Request Url
	requestUrl := fmt.Sprintf("%s/%s", serverUrl, request.Endpoint)

	additionalHeaders, body, err := request.buildPolicy(request)
	if err != nil {
		return res, fmt.Errorf("error executing buildpolicy: %s", err)
	}
	res, err = http.NewRequest(request.Method, requestUrl, body)
	if err != nil {
		return res, fmt.Errorf("error creating new request")
	}

	q := res.URL.Query()
	for key, val := range request.QueryParams {
		stringVal, err := util.GetStringFromInterface(val)
		if err != nil {
			return res, fmt.Errorf("error GetStringFromInterface: %s", err)
		}
		q.Add(key, stringVal)
	}
	res.URL.RawQuery = q.Encode()

	for key, val := range request.Headers {
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
	if request.BodyType == "multipart" {
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
