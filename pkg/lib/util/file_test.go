package util

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/spf13/afero"
)

type testOpenFileStruct struct {
	filename string
	expError error
	expHash  [16]byte
}

func TestOpenFileOrUrl(t *testing.T) {
	filesystem.Fs = afero.NewMemMapFs()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/exists" {
			fmt.Fprint(w, "Hallo ich bin online!")
		} else {
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	err := afero.WriteFile(filesystem.Fs, "localExists", []byte("Hallo ich bin da!"), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	tests := []testOpenFileStruct{
		{
			filename: "localExists",
			expError: nil,
			expHash:  [16]byte{41, 141, 109, 196, 242, 228, 71, 53, 148, 161, 107, 123, 254, 212, 41, 76},
		},
		{
			filename: "localNotExists",
			expError: fmt.Errorf("open localNotExists: file does not exist"),
			expHash:  [16]byte{},
		},
		{
			filename: ts.URL + "/exists",
			expError: nil,
			expHash:  [16]byte{178, 167, 33, 73, 246, 252, 206, 144, 244, 63, 78, 83, 51, 123, 185, 162},
		},
		{
			filename: ts.URL + "/notExists",
			expError: fmt.Errorf("StatusCode of requests file '%s/notExists' is 404", ts.URL),
			expHash:  [16]byte{},
		},
	}

	for _, v := range tests {
		t.Run(v.filename, func(t *testing.T) {
			file, err := OpenFileOrUrl(v.filename, "")
			if err != nil {
				if err.Error() != v.expError.Error() {
					t.Errorf("Got '%s' != '%s' Exp", err, v.expError)
				}
			} else {
				defer file.Close()
				data, err := io.ReadAll(file)
				go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

				if md5.Sum(data) != v.expHash {
					t.Errorf("Got '%s' != '%s' Exp", md5.Sum(data), v.expHash)
				}
			}
		})
	}

}

func TestOpenLocalFile(t *testing.T) {
	filesystem.Fs = afero.NewMemMapFs()

	err := afero.WriteFile(filesystem.Fs, filepath.Join("/", "root", "file.json"), []byte("From ROOT /"), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "file.json", []byte("From binary ./"), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, filepath.Join("/", "manifestdir", "file.json"), []byte("From manifest"), 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	reader, err := openLocalFile("/root/file.json", "/manifestdir")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	defer reader.Close()
	rootFile, err := io.ReadAll(reader)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if string(rootFile) != "From ROOT /" {
		t.Errorf("Wrong file content for root file: %s", string(rootFile))
	}

	reader, err = openLocalFile("file.json", "/manifestdir")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	defer reader.Close()
	manifestFile, err := io.ReadAll(reader)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if string(manifestFile) != "From manifest" {
		t.Errorf("Wrong file content for manifest file: %s", string(manifestFile))
	}

	reader, err = openLocalFile("./file.json", "/manifestdir")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	defer reader.Close()

	binaryFile, err := io.ReadAll(reader)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if string(binaryFile) != "From binary ./" {
		t.Errorf("Wrong file content for binary file: %s", string(binaryFile))
	}
}

func errorStringIfNotNil(err error) (errS string) {
	if err == nil {
		return ""
	}
	return err.Error()
}
