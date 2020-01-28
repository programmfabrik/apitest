package filesystem

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/afero"
)

func TestMemFS(t *testing.T) {
	Fs = afero.NewMemMapFs()

	for i := 0; i < 1000; i++ {
		Fs.MkdirAll(filepath.Join("store", "test1", "data", strconv.Itoa(i)), 755)
	}
	for i := 0; i < 1000; i++ {
		_, err := Fs.Open(filepath.Join("store", "test1", "data", strconv.Itoa(i)))
		if err != nil {
			t.Error(err)
		}
	}
}
