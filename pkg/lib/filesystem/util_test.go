package filesystem

import (
	"fmt"
	"github.com/spf13/afero"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCopyFileBetweenFileSystems(t *testing.T) {
	outFS := afero.NewMemMapFs()
	inFS := afero.NewMemMapFs()

	inPath := filepath.Join(os.TempDir(), "tmpfile.tmp")
	outPath := filepath.Join(afero.GetTempDir(inFS, ""), "tmpfile.tmp")

	err := CopyFileBetweenFileSystems(inFS, outFS, inPath, outPath, 644)
	if err.Error() != fmt.Sprintf("open %s: file does not exist", inPath) {
		t.Errorf("Unexpected Error: %s. Expected: %s", err, fmt.Sprintf("open %s: file does not exist", inPath))
	}

	inFS.Create(inPath)

	err = CopyFileBetweenFileSystems(inFS, outFS, inPath, outPath, 644)
	if err != nil {
		t.Errorf("Could not copy file: %s", err)
	}

	//Check if the files are the same
	fileIn, _ := inFS.OpenFile(inPath, os.O_RDONLY, 644)
	fileOut, _ := outFS.OpenFile(outPath, os.O_RDONLY, 644)

	fileInContent, _ := ioutil.ReadAll(fileIn)
	fileOutContent, _ := ioutil.ReadAll(fileOut)

	if string(fileInContent) != string(fileOutContent) {
		t.Error("InFile != OutFile")
	}

}

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
