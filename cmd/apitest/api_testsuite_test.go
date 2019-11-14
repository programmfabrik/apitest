package main

import (
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	"github.com/spf13/afero"
)

func TestLoadManifest(t *testing.T) {
	filesystem.Fs = afero.NewMemMapFs()

	afero.WriteFile(filesystem.Fs, "externalFile", []byte(`{"load":{"me":"loaded"}}`), 644)

	afero.WriteFile(filesystem.Fs, "testManifest.json", []byte(`{"testload": {{ file "externalFile" | qjson "load.me"}}}`), 644)

	s := Suite{manifestPath: "testManifest.json"}

	res, err := s.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	if string(res) != `{"testload": "loaded"}` {
		t.Errorf(`Exp '{"testload": "loaded"}' != '%s' Got`, res)
	}

}
