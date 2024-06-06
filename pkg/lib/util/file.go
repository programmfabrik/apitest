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
func OpenFileOrUrl(path, rootDir string) (string, io.ReadCloser, error) {
	pathSpec, ok := ParsePathSpec(path)
	if ok {
		path = pathSpec.Path
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		io, err := openRemoteFile(path)
		return path, io, err
	} else {
		io, err := openLocalFile(path, rootDir)
		return path, io, err
	}
}

func openRemoteFile(absPath string) (io.ReadCloser, error) {
	resp, err := c.Get(absPath)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("StatusCode of requests file '%s' is %d", absPath, resp.StatusCode)
	}
	return resp.Body, err
}

func openLocalFile(path, rootDir string) (io.ReadCloser, error) {
	return filesystem.Fs.Open(LocalPath(path, rootDir))
}

func LocalPath(path, rootDir string) string {
	var absPath string
	if strings.HasPrefix(path, "./") {
		//Path relative to binary
		absPath = path
	} else if strings.HasPrefix(path, "/") {
		//Absolute Path
		absPath = filepath.Join("/", path)
	} else {
		absPath = filepath.Join(rootDir, path)
	}
	return absPath
}
