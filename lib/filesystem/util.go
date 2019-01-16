package filesystem

import (
	"github.com/spf13/afero"
	"io/ioutil"
	"os"
	"path/filepath"
)

func CopyFileBetweenFileSystems(inFS, outFS afero.Fs, pathIn, pathOut string, perm os.FileMode) (err error) {
	fileIn, err := inFS.OpenFile(pathIn, os.O_RDONLY, perm)
	if err != nil {
		return
	}
	defer fileIn.Close()
	fileInContent, err := ioutil.ReadAll(fileIn)
	if err != nil {
		return
	}

	err = os.MkdirAll(filepath.Dir(pathOut), os.ModePerm)
	if err != nil {
		return
	}
	fileOut, err := outFS.OpenFile(pathOut, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return
	}
	defer fileOut.Close()

	_, err = fileOut.Write(fileInContent)

	return
}
