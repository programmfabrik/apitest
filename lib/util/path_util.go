package util

import (
	"path/filepath"
	"strings"
)

/*
throughout this file we assume 'manifestDir' to be an absolute path
*/

func GetAbsPath(manifestDir, pathSpec string) string {
	return filepath.Join(manifestDir, pathSpec[1:])
}

func IsPathSpec(pathSpec string) bool {
	return strings.HasPrefix(pathSpec, "@")
}
