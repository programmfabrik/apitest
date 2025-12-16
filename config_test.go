package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	go_test_utils "github.com/programmfabrik/go-test-utils"
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
	// Setup testserver
	server = go_test_utils.NewTestServer(go_test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
		"/api/v1/settings": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"db-name\": \"sTest\"}"))
		},
	})

	// Setup test filesystem
	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath1), 0755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath2), 0755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath3), 0755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath4), 0755)
	filesystem.Fs.MkdirAll(filepath.Dir(manifestPath5), 0755)
	filesystem.Fs.MkdirAll(filepath.Join("path", "empty"), 0755)

	afero.WriteFile(filesystem.Fs, manifestPath1, []byte(""), 0644)
	afero.WriteFile(filesystem.Fs, manifestPath2, []byte(""), 0644)
	afero.WriteFile(filesystem.Fs, manifestPath3, []byte(""), 0644)
	afero.WriteFile(filesystem.Fs, manifestPath4, []byte(""), 0644)
	afero.WriteFile(filesystem.Fs, manifestPath5, []byte(""), 0644)

}

func TestTestToolConfig_ExtractTestDirectories(t *testing.T) {
	SetupFS()

	// Invalid rootDirectory -> Expect error
	_, err := newTestToolConfig(server.URL+"/api/v1", []string{"invalid"}, false, false, false)
	go_test_utils.ExpectError(t, err, "NewTestToolConfig did not fail on invalid root directory")

	// Invalid rootDirectory -> Expect error
	conf, err := newTestToolConfig(server.URL+"/api/v1", []string{"path"}, false, false, false)
	go_test_utils.ExpectNoError(t, err, "NewTestToolConfig did fail on valid root directory")

	expectedResults := []string{
		filepath.Dir(manifestPath1),
		filepath.Dir(manifestPath2),
		filepath.Dir(manifestPath3),
	}

	if len(expectedResults) != len(conf.testDirectories) {
		t.Errorf("Len: Got %d, expected %d", len(conf.testDirectories), len(expectedResults))
	}

	for k, v := range expectedResults {
		if conf.testDirectories[k] != v {
			t.Errorf("Got %s, exptected != %s", conf.testDirectories[k], v)
		}
	}

}
