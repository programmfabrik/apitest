package api

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/programmfabrik/fylr-apitest/lib/test_utils"

	"github.com/spf13/afero"
)

func TestBuildMultipart(t *testing.T) {
	assertContent := "mock"
	assertFilename := "path/mockfile.json"
	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll("test/path", 0755)
	afero.WriteFile(filesystem.Fs, fmt.Sprintf("test/%s", assertFilename), []byte(assertContent), 0644)

	testRequest := Request{
		Body: map[string]interface{}{
			"somekey": fmt.Sprintf("@%s", assertFilename),
		},
		ManifestDir: "test/",
		BodyType:    "multipart",
	}

	httpRequest, err := testRequest.buildHttpRequest("some_interface", "some_token")
	test_utils.CheckError(t, err, "error building multipart request")

	testReader, err := httpRequest.MultipartReader()
	test_utils.CheckError(t, err, "error getting multipart reader from request")
	part, err := testReader.NextPart()
	test_utils.CheckError(t, err, "error reading part from multipart reader")

	test_utils.AssertStringEquals(t, part.FileName(), assertFilename)
	buf := new(bytes.Buffer)
	buf.ReadFrom(part)
	test_utils.AssertStringEquals(t, assertContent, buf.String())
}

func TestBuildMultipart_ErrPathSpec(t *testing.T) {
	testRequest := Request{
		Body: map[string]interface{}{
			"somekey": fmt.Sprintf("noPathspec"),
		},
		ManifestDir: "test/path/",
	}

	_, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pathSpec noPathspec is not valid") {
		t.Error("expected error because of invalid pathspec")
	}
}

func TestBuildMultipart_ErrPathSpecNoString(t *testing.T) {
	testRequest := Request{
		Body: map[string]interface{}{
			"somekey": 1,
		},
		ManifestDir: "test/path/",
	}

	_, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pathSpec should be a string") {
		t.Error("expected error because of invalid type for pathSpec")
	}
}

func TestBuildMultipart_FileDoesNotExist(t *testing.T) {
	testRequest := Request{
		Body: map[string]interface{}{
			"somekey": "@does_not_exist.json",
		},
		ManifestDir: "test/path/",
	}

	_, _, err := buildMultipart(testRequest)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does_not_exist.json: file does not exist") {
		t.Errorf("expected error because file does not exist")
	}
}
