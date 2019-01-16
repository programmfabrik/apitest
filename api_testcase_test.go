package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/api"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/spf13/afero"
)

func TestCollectResponseShouldWork(t *testing.T) {

	i := -2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"ID":%d},{"ID":%d}]`, i, i+1)
		i = i + 2
	}))
	defer ts.Close()

	testManifest := []byte(`
        {
            "name": "CollectTest",
				"request":{
				"endpoint": "suggest", 
				"method": "GET"
				},
		        "timeout_ms":3000,
		        "collect_response":[
					{
						"body":[{
							"ID":2
						}]
					},
					{
						"body":[{
							"ID":22
						}]
					},
					{
						"body":[{
							"ID":122
						}]
					},
					{
						"body":[{
							"ID":212
						}]
					}
				]
			
        }
`)

	filesystem.Fs = afero.NewMemMapFs()
	afero.WriteFile(filesystem.Fs, "manifest.json", []byte(testManifest), 644)

	r := report.NewReport()

	var test Case
	err := json.Unmarshal(testManifest, &test)
	if err != nil {
		t.Fatal(err)
	}
	test.reporter = r
	test.session, _ = api.NewSession(ts.URL, &http.Client{}, &api.SessionAuthentication{}, api.NewStore())

	test.runAPITestCase()

	testResult := string(r.GetTestResult(report.ParseJSONResult))

	if r.DidFail() {
		t.Errorf("collectResponse did not work: %s", testResult)
	}

}

func TestCollectLoadExternalFile(t *testing.T) {

	i := -2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"ID":%d},{"ID":%d}]`, i, i+1)
		i = i + 2
	}))
	defer ts.Close()

	filesystem.Fs = afero.NewMemMapFs()
	externalFile := []byte(`{
						"body":[{
							"ID":2
						}]
					}`)

	testManifest := []byte(`
        {
            "name": "CollectTest",
				"request":{
				"endpoint": "suggest", 
				"method": "GET"
				},
		        "timeout_ms":300,
		        "collect_response":["@collect.json"]
        }`)

	filesystem.Fs = afero.NewMemMapFs()
	afero.WriteFile(filesystem.Fs, "manifest.json", []byte(testManifest), 644)
	afero.WriteFile(filesystem.Fs, "collect.json", []byte(externalFile), 644)

	r := report.NewReport()

	var test Case
	err := json.Unmarshal(testManifest, &test)
	if err != nil {
		t.Fatal(err)
	}
	test.reporter = r
	test.session, _ = api.NewSession(ts.URL, &http.Client{}, &api.SessionAuthentication{}, api.NewStore())

	test.runAPITestCase()

	testResult := string(r.GetTestResult(report.ParseJSONResult))

	if r.DidFail() {
		t.Errorf("collectResponse did not work: %s", testResult)
	}

}

func TestCollectLoadExternalCollect(t *testing.T) {

	i := -2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"ID":%d},{"ID":%d}]`, i, i+1)
		i = i + 2
	}))
	defer ts.Close()

	filesystem.Fs = afero.NewMemMapFs()
	externalFile := []byte(`[{
						"body":[{
							"ID":2
						}]
					},{
						"body":[{
							"ID":6
						}]
					},{
						"body":[{
							"ID":7
						}]
					}]`)

	testManifest := []byte(`
        {
            "name": "CollectTest",
				"request":{
				"endpoint": "suggest", 
				"method": "GET"
				},
		        "timeout_ms":3000,
		        "collect_response":"@collect.json"
        }`)

	filesystem.Fs = afero.NewMemMapFs()
	afero.WriteFile(filesystem.Fs, "manifest.json", []byte(testManifest), 644)
	afero.WriteFile(filesystem.Fs, "collect.json", []byte(externalFile), 644)

	r := report.NewReport()

	var test Case
	err := json.Unmarshal(testManifest, &test)
	if err != nil {
		t.Fatal(err)
	}
	test.reporter = r
	test.session, _ = api.NewSession(ts.URL, &http.Client{}, &api.SessionAuthentication{}, api.NewStore())

	test.runAPITestCase()

	testResult := string(r.GetTestResult(report.ParseJSONResult))

	if r.DidFail() {
		t.Errorf("collectResponse did not work: %s", testResult)
	}

}

func TestCollectEvents(t *testing.T) {
	i := 115
	j := 100
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 961,
            "object_version": 1,
            "object_id": 118,
            "schema": "USER",
            "objecttype": "pictures",
            "global_object_id": "118@8367e587-f999-4e72-b69d-b5742eb4d5f4",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 962,
            "object_version": 0,
            "object_id": 1000832836,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 963,
            "object_version": 0,
            "object_id": %d,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    },
    {
        "event": {
            "type": "OBJECT_INDEX",
            "_id": 963,
            "object_version": 0,
            "object_id": %d,
            "schema": "BASE",
            "basetype": "asset",
            "timestamp": "2018-11-28T17:37:27+01:00",
            "pollable": true
        }
    }
]
`, i, j)
		i++
		j--
	}))
	defer ts.Close()

	filesystem.Fs = afero.NewMemMapFs()
	externalFile := []byte(`[{"body":[{"event":{"object_id":117}}]},{"body":[{"event":{"object_id":118}}]},{"body":[{"event":{"object_id":418}}]},{"body":[{"event":{"object_id":92}}]}]`)

	testManifest := []byte(`
        {
            "name": "CollectTest",
				"request":{
				"endpoint": "suggest", 
				"method": "GET"
				},
		        "timeout_ms":6000,
		        "collect_response":"@collect.json"
        }`)

	filesystem.Fs = afero.NewMemMapFs()
	afero.WriteFile(filesystem.Fs, "manifest.json", []byte(testManifest), 644)
	afero.WriteFile(filesystem.Fs, "collect.json", []byte(externalFile), 644)

	r := report.NewReport()

	var test Case
	err := json.Unmarshal(testManifest, &test)
	if err != nil {
		t.Fatal(err)
	}
	test.reporter = r
	test.session, _ = api.NewSession(ts.URL, &http.Client{}, &api.SessionAuthentication{}, api.NewStore())

	test.runAPITestCase()

	testResult := string(r.GetTestResult(report.ParseJSONResult))

	if r.DidFail() {
		t.Errorf("collectResponse did not work: %s", testResult)
	}

}

func TestCollectResponseShouldFail(t *testing.T) {

	i := 2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"ID":%d},{"ID":%d}]`, i, i+1)
		i = i + 2
	}))
	defer ts.Close()

	testManifest := []byte(`
        {
            "name": "CollectTest",
				"request":{
				"endpoint": "suggest", 
				"method": "GET"
				},
		        "timeout_ms":30,
		        "collect_response":[
					{
						"body":[{
							"ID":1
						}]
					}
				]
			
        }
`)

	filesystem.Fs = afero.NewMemMapFs()
	afero.WriteFile(filesystem.Fs, "manifest.json", []byte(testManifest), 644)

	r := report.NewReport()

	var test Case
	err := json.Unmarshal(testManifest, &test)
	if err != nil {
		t.Fatal(err)
	}
	test.reporter = r
	test.session, _ = api.NewSession(ts.URL, &http.Client{}, &api.SessionAuthentication{}, api.NewStore())

	test.runAPITestCase()

	r.GetTestResult(report.ParseJSONResult)

	log := r.GetLog()

	if !r.DidFail() {
		t.Errorf("Did not fail but it should")
	}

	if len(log) != 2 {
		t.Errorf("Length if log != 2")
	}

	if log[0] != "Pull Timeout '30ms' exceeded" {
		t.Errorf("Expected 'Pull Timeout '30ms' exceeded' != '%s' Got", log[0])
	}

	if log[1] != `Collect response not found: {"body":[{"ID":1}]}` {
		t.Errorf(`Expected 'Collect response not found: {"body":[{"ID":1}]}' != '%s' Got`, log[1])
	}
}
