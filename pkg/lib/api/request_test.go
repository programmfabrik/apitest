package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func TestRequestBuildHttp(t *testing.T) {
	request := Request{
		Endpoint: "endpoint",
		Method:   "DO!",
		QueryParams: map[string]any{
			"query_param": "value",
		},
		ServerURL: "serverUrl",
	}
	request.buildPolicy = func(request Request) (ah map[string]string, b io.Reader, err error) {
		ah = make(map[string]string)
		ah["mock-header"] = "application/mock"
		b = strings.NewReader("mock_body")
		return ah, b, nil
	}
	httpRequest, err := request.buildHttpRequest()
	go_test_utils.ExpectNoError(t, err, fmt.Errorf("error building http-request: %w", err).Error())
	go_test_utils.AssertStringEquals(t, httpRequest.Header.Get("mock-header"), "application/mock")

	assertBody, err := io.ReadAll(httpRequest.Body)
	go_test_utils.ExpectNoError(t, err, fmt.Errorf("error reading http-request body: %w", err).Error())
	go_test_utils.AssertStringEquals(t, string(assertBody), "mock_body")

	url := httpRequest.URL
	go_test_utils.AssertStringEquals(t, url.RawQuery, "query_param=value")
	go_test_utils.AssertStringEquals(t, url.Path, "serverUrl/endpoint")
}

func TestBuildCurl(t *testing.T) {
	request := Request{
		Endpoint: "endpoint",
		Method:   "GET",
		QueryParams: map[string]any{
			"query_param": "value",
		},
		ServerURL: "https://serverUrl",
		Body: jsutil.Object{
			"hey": 1,
		},
	}

	// exp := `curl \
	// 	-X 'GET' \
	// 	-d '{"hey":1}' \
	// 	-H 'Content-Type: application/json' \
	// 	-H 'User-Agent: ' \
	// 	'https://serverUrl/endpoint?query_param=value'`

	exp := `curl -X 'GET' -d '{"hey":1}' -H 'Content-Type: application/json' -H 'User-Agent: ' 'https://serverUrl/endpoint?query_param=value'`

	if request.ToString(true) != exp {
		t.Fatalf("Did not match right curl command. Expected '%s' != '%s' GOT", exp, request.ToString(true))
	}
}

func TestRequestBuildHttpWithCookie(t *testing.T) {
	reqCookies := map[string]*requestCookie{
		"sess": {
			ValueFromStore: "sess_cookie",
		},
		"sess2": {
			ValueFromStore: "?sess2_cookie",
			Value:          "you_sec_sess",
		},
	}
	storeCookies := map[string]http.Cookie{
		"sess_cookie": {
			Value: "your_session",
		},
	}
	request := Request{
		Endpoint: "dummy",
		Method:   "GET",
		Cookies:  reqCookies,
	}
	request.buildPolicy = buildRegular
	ds := datastore.NewStore(false)
	for key, val := range storeCookies {
		err := ds.Set(key, val)
		if err != nil {
			t.Fatalf("Could not store cookie in datastore: %s", err.Error())
		}
	}
	request.DataStore = ds
	httpRequest, err := request.buildHttpRequest()
	if err != nil {
		t.Fatalf("Could not build http request: %s", err.Error())
	}
	for k, v := range reqCookies {
		ck, err := httpRequest.Cookie(k)
		if err != nil {
			t.Fatalf("Could not retieve cookie '%s' from http request: %s", k, err.Error())
		}
		if v.Value != "" && ck.Value != v.Value {
			t.Fatalf("Cookie %s value: '%s', expected: '%s'", k, ck.Value, v.Value)
		}
		if v.ValueFromStore != "" {
			st := v.ValueFromStore
			if strings.HasPrefix(v.ValueFromStore, "?") {
				st = st[1:]
			}
			st = fmt.Sprintf("%s_cookie", st)
			sCk, ok := storeCookies[st]
			if !ok {
				continue
			}
			if v.Value == "" && ck.Value != sCk.Value {
				t.Fatalf("Cookie %s value: '%s', expected: '%s'", k, ck.Value, sCk.Value)
			}
		}
	}
}
