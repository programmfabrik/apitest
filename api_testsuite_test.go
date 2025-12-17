package main

import (
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/spf13/afero"
)

func TestLoadManifest(t *testing.T) {
	filesystem.Fs = afero.NewMemMapFs()

	err := afero.WriteFile(filesystem.Fs, "externalFile", []byte(`{"load":{"me":"loaded"}}`), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "testManifest.json", []byte(`{"testload": {{ file "externalFile" | gjson "load.me"}}}`), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	s := Suite{manifestPath: "testManifest.json"}

	res, err := s.loadManifest()
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	if string(res) != `{"testload": "loaded"}` {
		t.Errorf(`Exp '{"testload": "loaded"}' != '%s' Got`, res)
	}
}

func TestLoadManifestCustomDelimiters(t *testing.T) {
	filesystem.Fs = afero.NewMemMapFs()

	err := afero.WriteFile(filesystem.Fs, "externalFile", []byte(`{"load":{"me":"loaded"}}`), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	afero.WriteFile(filesystem.Fs, "testManifest.json", []byte(`// template-delims: ## ##
		// template-remove-tokens: "<placeholder>" "...."
	{"testload": ## file "externalFile" | gjson "load.me" ##}"...."`), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	s := Suite{manifestPath: "testManifest.json"}
	res, err := s.loadManifest()
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	if string(res) != `

	{"testload": "loaded"}` {
		t.Errorf(`Exp '{"testload": "loaded"}' != '%s' Got`, res)
	}
}
