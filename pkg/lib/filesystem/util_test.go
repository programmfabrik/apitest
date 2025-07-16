package filesystem

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/afero"
)

func TestMemFS(t *testing.T) {
	Fs = afero.NewMemMapFs()

	for i := range 1000 {
		Fs.MkdirAll(filepath.Join("store", "test1", "data", strconv.Itoa(i)), 0755)
	}
	for i := range 1000 {
		_, err := Fs.Open(filepath.Join("store", "test1", "data", strconv.Itoa(i)))
		if err != nil {
			t.Error(err)
		}
	}
}
