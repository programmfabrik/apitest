package api

import (
	"strings"
	"testing"

	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func TestResponse_ToGenericJson(t *testing.T) {
	response := Response{
		statusCode: 200,
		headers: map[string][]string{
			"foo": {"bar"},
		},
	}
	genericJson, err := response.ToGenericJSON()
	go_test_utils.ExpectNoError(t, err, "error calling response.ToGenericJson")

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
	go_test_utils.ExpectNoError(t, err, "unexpected error")
	go_test_utils.AssertIntEquals(t, response.statusCode, responseSpec.StatusCode)
	go_test_utils.AssertStringEquals(t, response.headers["foo"][0], "bar")
}

func TestResponse_NewResponseFromSpec_StatusCode_not_set(t *testing.T) {
	responseSpec := ResponseSerialization{
		Body: nil,
	}
	response, err := NewResponseFromSpec(responseSpec)
	go_test_utils.ExpectNoError(t, err, "unexpected error")
	go_test_utils.AssertIntEquals(t, response.statusCode, 200)
}

func TestResponse_NewResponse(t *testing.T) {
	response, err := NewResponse(200, nil, strings.NewReader("foo"), nil, ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, "unexpected error")
	go_test_utils.AssertIntEquals(t, response.statusCode, 200)
}

func TestResponse_String(t *testing.T) {
	response, err := NewResponse(200, nil, strings.NewReader("{\"foo\": \"bar\"}"), nil, ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, "error constructing response")
	assertString := "200\n\n\n{\n  \"foo\": \"bar\"\n}"
	go_test_utils.AssertStringEquals(t, response.ToString(), assertString)
}
