package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/programmfabrik/go-test-utils"
	"github.com/spf13/afero"
)

var (
	server *httptest.Server

	manifestPath1 = filepath.Join("path", "contain", "manifest.json")
	manifestPath2 = filepath.Join("path", "contain2", "inner_contain", "manifest.json")
	manifestPath3 = filepath.Join("path", "contain2", "inner_contain2", "manifest.json")
	manifestPath4 = filepath.Join("noPath", "contain2", "inner_contain2", "manifest.json")
	manifestPath5 = filepath.Join("path", "noManifest/NOmanifest.yaml")
)

func SetupFS() {
	//Setup testserver
	server = test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
		"/api/v1/settings": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"db-name\": \"sTest\"}"))
		},
	})

	//Setup test filesystem
	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath1), 755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath2), 755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath3), 755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath4), 755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath5), 755)
	filesystem.Fs.MkdirAll(filepath.Join("path", "empty"), 755)

	afero.WriteFile(filesystem.Fs, manifestPath1, []byte(""), 644)
	afero.WriteFile(filesystem.Fs, manifestPath2, []byte(""), 644)
	afero.WriteFile(filesystem.Fs, manifestPath3, []byte(""), 644)
	afero.WriteFile(filesystem.Fs, manifestPath4, []byte(""), 644)
	afero.WriteFile(filesystem.Fs, manifestPath5, []byte(""), 644)

}

func TestTestToolConfig_ExtractTestDirectories(t *testing.T) {
	SetupFS()

	//Invalid rootDirectory -> Expect error
	_, err := NewTestToolConfig(server.URL+"/api/v1", "sTest", []string{"invalid"})
	test_utils.ExpectError(t, err, "NewTestToolConfig did not fail on invalid root directory")

	//Invalid rootDirectory -> Expect error
	conf, err := NewTestToolConfig(server.URL+"/api/v1", "sTest", []string{"path"})
	test_utils.CheckError(t, err, "NewTestToolConfig did fail on valid root directory")

	expectedResults := []string{
		filepath.Dir(manifestPath1),
		filepath.Dir(manifestPath2),
		filepath.Dir(manifestPath3),
	}

	if len(expectedResults) != len(conf.TestDirectories) {
		t.Errorf("Len: Got %d != %d Expected", len(conf.TestDirectories), len(expectedResults))
	}

	for k, v := range expectedResults {
		if conf.TestDirectories[k] != v {
			t.Errorf("Got %s != %s Expected", conf.TestDirectories[k], v)
		}
	}

}
