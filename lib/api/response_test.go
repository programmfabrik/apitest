package api

import (
	"strings"
	"testing"

	"github.com/programmfabrik/go-test-utils"
)

func TestResponse_ToGenericJson(t *testing.T) {
	response := Response{
		statusCode: 200,
		headers: map[string][]string{
			"foo": {"bar"},
		},
	}
	genericJson, err := response.ToGenericJson()
	test_utils.CheckError(t, err, "error calling response.ToGenericJson")

	jsonObjResp, ok := genericJson.(map[string]interface{})
	if !ok {
		t.Fatalf("responseJson should be object")
	}
	statusCode, ok := jsonObjResp["statuscode"]
	if !ok {
		t.Fatalf("responseJsonObj should have status code field")
	}
	if statusCode != float64(200) {
		t.Errorf("responseJson had wrong statuscode, expected 200, got: %d", statusCode)
	}
	jsonHeaders, ok := jsonObjResp["header"]
	if !ok {
		t.Fatalf("responseJsonObj should have headers")
	}
	headersMap, ok := jsonHeaders.(map[string]interface{})
	if !ok {
		t.Fatalf("headers should be map")
	}
	if headersMap["foo"].([]interface{})[0].(string) != "bar" {
		t.Errorf("expected foo header to be bar")
	}
}

func TestResponse_NewResponseFromSpec(t *testing.T) {
	responseSpec := ResponseSerialization{
		StatusCode: 200,
		Headers: map[string][]string{
			"foo": {"bar"},
		},
		Body: nil,
	}
	response, err := NewResponseFromSpec(responseSpec)
	test_utils.CheckError(t, err, "unexpected error")
	test_utils.AssertIntEquals(t, response.statusCode, responseSpec.StatusCode)
	test_utils.AssertStringEquals(t, response.headers["foo"][0], "bar")
}

func TestResponse_NewResponseFromSpec_StatusCode_not_set(t *testing.T) {
	responseSpec := ResponseSerialization{
		Body: nil,
	}
	response, err := NewResponseFromSpec(responseSpec)
	test_utils.CheckError(t, err, "unexpected error")
	test_utils.AssertIntEquals(t, response.statusCode, 200)
}

func TestResponse_NewResponse(t *testing.T) {
	response, err := NewResponse(200, nil, strings.NewReader("foo"))
	test_utils.CheckError(t, err, "unexpected error")
	test_utils.AssertIntEquals(t, response.statusCode, 200)
}

func TestResponse_String(t *testing.T) {
	response, err := NewResponse(200, nil, strings.NewReader("{\"foo\": \"bar\"}"))
	assertString := "200\n\n\n{\"foo\": \"bar\"}"
	test_utils.CheckError(t, err, "error constructing response")
	test_utils.AssertStringEquals(t, response.ToString(), assertString)
}
