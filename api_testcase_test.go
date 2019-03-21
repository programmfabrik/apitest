package main

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/api"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"github.com/spf13/afero"
)

func TestQjson(t *testing.T) {
	jsolo := `{"body":[{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":46,"global_object_id":"1@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":1,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:05+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":47,"global_object_id":"2@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":2,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:05+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":48,"global_object_id":"3@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":3,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":49,"global_object_id":"4@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":4,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"event":{"_id":50,"global_object_id":"1@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":1,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":51,"global_object_id":"2@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":2,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INDEX"}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":52,"global_object_id":"5@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":5,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":53,"global_object_id":"6@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":6,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":54,"global_object_id":"7@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":7,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:06+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":55,"global_object_id":"8@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":8,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":56,"global_object_id":"9@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":9,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":57,"global_object_id":"10@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":10,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":58,"global_object_id":"11@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":11,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":59,"global_object_id":"12@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":12,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"event":{"_id":60,"global_object_id":"5@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":5,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":61,"global_object_id":"6@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":6,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":62,"global_object_id":"3@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":3,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":63,"global_object_id":"4@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":4,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INDEX"}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":64,"global_object_id":"13@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":13,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:07+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":65,"global_object_id":"14@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":14,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":66,"global_object_id":"15@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":15,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"_session":{"token":"ac554a02-3ef0-42da-8ffb-603d73de95f9"},"event":{"_id":67,"global_object_id":"16@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":16,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","session_self":true,"timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INSERT"},"user":{"_generated_displayname":"Root","_id":1}},{"event":{"_id":68,"global_object_id":"8@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":8,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":69,"global_object_id":"9@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":9,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INDEX"}},{"event":{"_id":70,"global_object_id":"7@ebe5e467-4da9-4cff-81b6-cee9b1385b7c","object_id":7,"object_version":1,"objecttype":"main","pollable":true,"schema":"USER","timestamp":"2019-03-13T10:41:08+01:00","type":"OBJECT_INDEX"}}],"header":{"Cache-Control":["no-cache"],"Content-Type":["application/json; charset=utf-8"],"Date":["Wed, 13 Mar 2019 09:41:16 GMT"],"Last-Modified":["Wed, 13 Mar 2019, 09:41:16 GMT"],"Pragma":["no-cache"],"Server":["Apache/2.4.25 (Debian)"],"Vary":["Origin,Accept-Encoding"],"X-Easydb-Api-Version":["1"],"X-Easydb-Base-Schema-Version":["207"],"X-Easydb-Solution":["simon"],"X-Easydb-User-Schema-Version":["2"]},"statuscode":200}`

	fmt.Println(gjson.Get(jsolo, "body|@reverse|0.event._id"))
}

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
	test.ServerURL = ts.URL
	test.dataStore = api.NewStore()

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
	test.ServerURL = ts.URL
	test.dataStore = api.NewStore()

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
	test.ServerURL = ts.URL
	test.dataStore = api.NewStore()

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
	test.ServerURL = ts.URL
	test.dataStore = api.NewStore()

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
	test.ServerURL = ts.URL
	test.dataStore = api.NewStore()

	test.runAPITestCase()

	r.GetTestResult(report.ParseJSONResult)

	log := r.GetLog()

	if !r.DidFail() {
		t.Errorf("Did not fail but it should")
	}

	if len(log) != 2 {
		t.Fatalf("Length of log != 2. Log '%s'", log)
	}

	if log[0] != "Pull Timeout '30ms' exceeded" {
		t.Errorf("Expected 'Pull Timeout '30ms' exceeded' != '%s' Got", log[0])
	}

	if log[1] != `Collect response not found: {"body":[{"ID":1}]}` {
		t.Errorf(`Expected 'Collect response not found: {"body":[{"ID":1}]}' != '%s' Got`, log[1])
	}
}

func TestHeaderFromDatastoreWithMap(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"Auth": "%s"}`, r.Header.Get("AuthHeader"))
	}))
	defer ts.Close()

	testManifest := []byte(`
        {
            "name": "CollectTest",
			"request":{
				"endpoint": "suggest", 
				"method": "GET",
				"header_from_store":{
					"authHeader":"hallo[du]"
				}
			},
			"response":{
				"body": {
					"Auth": "du index"
				}
			}
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
	test.ServerURL = ts.URL

	test.dataStore = api.NewStore()
	test.dataStore.Set("hallo[du]", "du index")
	test.dataStore.Set("hallo[sie]", "sie index")

	test.runAPITestCase()

	r.GetTestResult(report.ParseJSONResult)
	if r.DidFail() {
		t.Errorf("Did fail but it should not")
	}

}

func TestHeaderFromDatastoreWithSlice(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"Auth": "%s"}`, r.Header.Get("AuthHeader"))
	}))
	defer ts.Close()

	testManifest := []byte(`
        {
            "name": "CollectTest",
			"request":{
				"endpoint": "suggest", 
				"method": "GET",
				"header_from_store":{
					"authHeader":"hallo[3]"
				}
			},
			"response":{
				"body": {
					"Auth": "es index"
				}
			}
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
	test.ServerURL = ts.URL

	test.dataStore = api.NewStore()
	test.dataStore.Set("hallo[]", "du index")
	test.dataStore.Set("hallo[]", "sie index")
	test.dataStore.Set("hallo[]", "er index")
	test.dataStore.Set("hallo[]", "es index")
	test.dataStore.Set("hallo[]", "mama index")

	test.runAPITestCase()

	r.GetTestResult(report.ParseJSONResult)
	if r.DidFail() {
		t.Errorf("Did fail but it should not")
	}

}
