package test_utils

import (
	"log"
	"net/http"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/programmfabrik/golib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	TestServer = go_test_utils.NewTestServer(go_test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
		"/api/v1/session/authenticate": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"authenticated\": \"true\"}"))
		},
		"/api/v1/settings/purge": func(w *http.ResponseWriter, r *http.Request) {
			(*w).WriteHeader(500)
		},
		"/api/v1/mock": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"mocked\": \"true\"}"))
		},
	})
	TestClient = TestServer.Client()
)

// AssertJsonStringEquals checks if two json strings are equal when minified
func AssertJsonStringEquals(t testing.TB, expected, got string) {

	var (
		expectedJson, gotJson        any
		expectedMinified, gotMinifed []byte
	)

	err := jsutil.UnmarshalString(expected, &expectedJson)
	if err != nil {
		t.Error(err)
	}
	expectedMinified, err = golib.JsonBytesIndent(expectedJson, "", "")
	if err != nil {
		log.Fatal(err)
	}

	err = jsutil.UnmarshalString(got, &gotJson)
	if err != nil {
		t.Error(err)
	}
	gotMinifed, err = golib.JsonBytesIndent(gotJson, "", "")
	if err != nil {
		log.Fatal(err)
	}

	if string(expectedMinified) != string(gotMinifed) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expected, got, false)
		t.Error(dmp.DiffPrettyText(diffs))
	}
}
