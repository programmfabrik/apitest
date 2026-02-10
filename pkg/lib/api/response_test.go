package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/programmfabrik/golib"
	"github.com/tidwall/gjson"
)

func TestResponse_ToGenericJson(t *testing.T) {
	response := Response{
		StatusCode: golib.IntRef(200),
		Headers: map[string]any{
			"foo": []string{"bar"},
		},
	}
	genericJson, err := response.ToGenericJSON()
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	jsonObjResp, ok := genericJson.(map[string]any)
	if !ok {
		t.Fatalf("responseJson should be object")
	}
	statusCode, ok := jsonObjResp["statuscode"]
	if !ok {
		t.Fatalf("responseJsonObj should have status code field")
	}
	if statusCode != jsutil.Number("200") {
		t.Errorf("responseJson had wrong statuscode, expected 200, got: %d", statusCode)
	}
	jsonHeaders, ok := jsonObjResp["header"]
	if !ok {
		t.Fatalf("responseJsonObj should have headers")
	}
	headersMap, ok := jsonHeaders.(map[string]any)
	if !ok {
		t.Fatalf("headers should be map")
	}
	if headersMap["foo"].([]any)[0].(string) != "bar" {
		t.Errorf("expected foo header to be bar")
	}
}

func TestResponse_NewResponseFromSpec(t *testing.T) {
	responseSpec := ResponseSerialization{
		StatusCode: golib.IntRef(200),
		Headers: map[string]any{
			"foo": []string{"bar"},
			"foo2:control": jsutil.Object{
				"must_not_exist": true,
			},
		},
		Body: nil,
	}
	response, err := NewResponseFromSpec(responseSpec)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertIntEquals(t, *response.StatusCode, *responseSpec.StatusCode)
	go_test_utils.AssertStringEquals(t, response.Headers["foo"].([]string)[0], "bar")
}

func TestResponse_NewResponseFromSpec_StatusCode_not_set(t *testing.T) {
	responseSpec := ResponseSerialization{
		Body: nil,
	}
	_, err := NewResponseFromSpec(responseSpec)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
}

func TestResponse_NewResponse(t *testing.T) {
	response, err := NewResponse(golib.IntRef(200), nil, nil, strings.NewReader("foo"), nil, ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertIntEquals(t, *response.StatusCode, 200)
}

func TestResponse_String(t *testing.T) {
	requestString := `{
		"foo": "bar",
		"foo2:control": {
			"must_not_exist": true
		}
	}`

	response, err := NewResponse(golib.IntRef(200), nil, nil, strings.NewReader(requestString), nil, ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	assertString := "200\n\n\n" + requestString
	assertString = strings.ReplaceAll(assertString, "\n", "")
	assertString = strings.ReplaceAll(assertString, " ", "")

	responseString := response.ToString()
	responseString = strings.ReplaceAll(responseString, "\n", "")
	responseString = strings.ReplaceAll(responseString, " ", "")

	go_test_utils.AssertStringEquals(t, responseString, assertString)
}

func TestResponse_Cookies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "sess",
			Value: "you_session_data",
		})
	}))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	defer res.Body.Close()

	header, err := httpHeaderToMap(res.Header)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	response, err := NewResponse(golib.IntRef(res.StatusCode), header, res.Cookies(), res.Body, nil, ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	jsonStr, err := response.ServerResponseToJsonString(false)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	if !gjson.Valid(jsonStr) {
		t.Fatalf("Invalid serialized JSON: %s", jsonStr)
	}

	v := gjson.Get(jsonStr, "cookie.sess")
	if !v.Exists() {
		t.Fatalf("No cookie found in JSON: %s", jsonStr)
	}
	if !v.IsObject() {
		t.Fatalf("Cookie raw object malformed in JSON: %s", jsonStr)
	}

	var ck http.Cookie
	vb, err := jsutil.Marshal(v.Value())
	go_test_utils.ExpectNoError(t, err, fmt.Sprintf("marshalling Cookie raw object: %v", v))
	err = jsutil.Unmarshal(vb, &ck)
	go_test_utils.ExpectNoError(t, err, fmt.Sprintf("unmarshalling into Cookie object: %v", v))

	go_test_utils.AssertStringEquals(t, ck.Value, "you_session_data")
}
