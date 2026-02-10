package filesystem

import (
	"path/filepath"
	"strconv"
	"testing"

	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/spf13/afero"
)

func TestMemFS(t *testing.T) {
	Fs = afero.NewMemMapFs()

	var err error

	for i := range 1000 {
		err = Fs.MkdirAll(filepath.Join("store", "test1", "data", strconv.Itoa(i)), 0755)
		go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	}
	for i := range 1000 {
		_, err := Fs.Open(filepath.Join("store", "test1", "data", strconv.Itoa(i)))
		go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	}
}

func errorStringIfNotNil(err error) (errS string) {
	if err == nil {
		return ""
	}
	return err.Error()
}
