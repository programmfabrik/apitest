package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
)

func TestRequestFromSpec(t *testing.T) {
	requestSpec := RequestSerialization{
		Endpoint: "session",
		Method:   "GET",
		QueryParams: map[string]interface{}{
			"priority":  "2",
			"format":    "long",
			"verbosity": 2,
			"object": map[string]interface{}{
				"test": 2,
			},
		},
		Headers: map[string]string{
			"testheader": "val",
		},
	}
	request := NewRequest(requestSpec, "manifestDir")
	test_utils.AssertStringEquals(t, request.queryParams["priority"], "2")
	test_utils.AssertStringEquals(t, request.queryParams["format"], "long")
	test_utils.AssertStringEquals(t, request.queryParams["verbosity"], "2")
	test_utils.AssertStringEquals(t, request.queryParams["object"], `{"test":2}`)
	test_utils.AssertStringEquals(t, request.headers["testheader"], "val")
	test_utils.AssertStringEquals(t, request.manifestDir, "manifestDir")
}

func TestRequestBuildHttp(t *testing.T) {
	request := Request{
		endpoint: "endpoint",
		method:   "DO!",
		queryParams: map[string]string{
			"query_param": "value",
		},
	}
	request.buildPolicy = func(request Request) (ah map[string]string, b io.Reader, err error) {
		ah = make(map[string]string)
		ah["mock-header"] = "application/mock"
		b = strings.NewReader("mock_body")
		return ah, b, nil
	}
	httpRequest, err := request.buildHttpRequest("serverUrl", "token")
	test_utils.CheckError(t, err, fmt.Sprintf("error building http-request: %s", err))
	test_utils.AssertStringEquals(t, httpRequest.Header.Get("mock-header"), "application/mock")
	test_utils.AssertStringEquals(t, httpRequest.Header.Get("x-easydb-token"), "token")

	assertBody, err := ioutil.ReadAll(httpRequest.Body)
	test_utils.CheckError(t, err, fmt.Sprintf("error reading http-request body: %s", err))
	test_utils.AssertStringEquals(t, string(assertBody), "mock_body")

	url := httpRequest.URL
	test_utils.AssertStringEquals(t, url.RawQuery, "query_param=value")
	test_utils.AssertStringEquals(t, url.Path, "serverUrl/api/v1/endpoint")
}

func TestRequestSetBuildPolicy_regular(t *testing.T) {
	request := Request{}
	request.setBuildPolicy("")
	if reflect.ValueOf(request.buildPolicy).Pointer() != reflect.ValueOf(buildRegular).Pointer() {
		t.Errorf("expected build policy to be 'buildRegular'")
	}
}

func TestRequestSetBuildPolicy_multipart(t *testing.T) {
	request := Request{}
	request.setBuildPolicy("multipart")
	if reflect.ValueOf(request.buildPolicy).Pointer() != reflect.ValueOf(buildMultipart).Pointer() {
		t.Errorf("expected build policy to be 'buildMultipart'")
	}
}

func TestRequestSetBuildPolicy_urlencoded(t *testing.T) {
	request := Request{}
	request.setBuildPolicy("urlencoded")
	if reflect.ValueOf(request.buildPolicy).Pointer() != reflect.ValueOf(buildUrlencoded).Pointer() {
		t.Errorf("expected build policy to be 'buildUrlencoded'")
	}
}
