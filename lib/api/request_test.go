package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
)

func TestRequestBuildHttp(t *testing.T) {
	request := Request{
		Endpoint: "endpoint",
		Method:   "DO!",
		QueryParams: map[string]interface{}{
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
	test_utils.AssertStringEquals(t, url.Path, "serverUrl/endpoint")
}
