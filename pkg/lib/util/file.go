package util

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
)

var c = &http.Client{
	Timeout: time.Second * 10,
}

// OpenFileOrUrl opens either a local file or gives the resp.Body from a remote file
func OpenFileOrUrl(path, rootDir string) (f io.ReadCloser, err error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return openRemoteFile(path)
	} else {
		return openLocalFile(path, rootDir)
	}
}

func openRemoteFile(absPath string) (f io.ReadCloser, err error) {
	var (
		resp *http.Response
	)

	resp, err = c.Get(absPath)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("StatusCode of requests file '%s' is %d", absPath, resp.StatusCode)
	}
	return resp.Body, err
}

func openLocalFile(path, rootDir string) (f io.ReadCloser, err error) {
	return filesystem.Fs.Open(LocalPath(path, rootDir))
}

func LocalPath(path, rootDir string) (absPath string) {
	if strings.HasPrefix(path, "./") {
		// Path relative to binary
		absPath = path
	} else if strings.HasPrefix(path, "/") {
		// Absolute Path
		absPath = filepath.Join("/", path)
	} else {
		absPath = filepath.Join(rootDir, path)
	}
	return absPath
}
