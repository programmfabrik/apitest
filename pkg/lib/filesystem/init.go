package filesystem

import "github.com/spf13/afero"

var Fs afero.Fs

func init() {
	Fs = afero.NewOsFs()
}
