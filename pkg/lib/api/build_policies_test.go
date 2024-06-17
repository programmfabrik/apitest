package api

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	go_test_utils "github.com/programmfabrik/go-test-utils"

	"github.com/spf13/afero"
)

func TestBuildMultipart(t *testing.T) {
	assertContent := "mock"
	assertFilename := "mockfile.json"
	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll("test/path", 0755)
	afero.WriteFile(filesystem.Fs, fmt.Sprintf("test/%s", assertFilename), []byte(assertContent), 0644)

	testRequest := Request{
		Body: map[string]any{
			"somekey": fmt.Sprintf("@%s", assertFilename),
		},
		ManifestDir: "test/",
		BodyType:    "multipart",
	}
	defer testRequest.Close()

	httpRequest, err := testRequest.buildHttpRequest()
	go_test_utils.ExpectNoError(t, err, "error building multipart request")

	testReader, err := httpRequest.MultipartReader()
	go_test_utils.ExpectNoError(t, err, "error getting multipart reader from request")
	part, err := testReader.NextPart()
	go_test_utils.ExpectNoError(t, err, "error reading part from multipart reader")

	go_test_utils.AssertStringEquals(t, part.FileName(), assertFilename)
	buf := new(bytes.Buffer)
	buf.ReadFrom(part)
	go_test_utils.AssertStringEquals(t, assertContent, buf.String())
}

func TestBuildMultipart_ErrPathSpec(t *testing.T) {
	testRequest := Request{
		Body: map[string]any{
			"somekey": fmt.Sprintf("noPathspec"),
		},
		ManifestDir: "test/path/",
	}

	_, _, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pathSpec noPathspec is not valid") {
		t.Error("expected error because of invalid pathspec")
	}
}

func TestBuildMultipart_ErrPathSpecNoString(t *testing.T) {
	testRequest := Request{
		Body: map[string]any{
			"somekey": 1,
		},
		ManifestDir: "test/path/",
	}

	_, _, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pathSpec should be a string") {
		t.Error("expected error because of invalid type for pathSpec")
	}
}

func TestBuildMultipart_FileDoesNotExist(t *testing.T) {
	testRequest := Request{
		Body: map[string]any{
			"somekey": "@does_not_exist.json",
		},
		ManifestDir: "test/path/",
	}

	_, _, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does_not_exist.json: file does not exist") {
		t.Errorf("expected error because file does not exist")
	}
}
