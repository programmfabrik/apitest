package util

import (
	"path/filepath"
)

/*
throughout this file we assume 'manifestDir' to be an absolute path
*/

func GetAbsPath(manifestDir, pathSpec string) string {
	return filepath.Join(manifestDir, pathSpec[1:])
}

func IsPathSpec(pathSpec []byte) bool {
	if len(pathSpec) < 3 {
		return false
	}

	if rune(pathSpec[0]) == rune('@') || rune(pathSpec[1]) == rune('@') {
		return true
	}
	if (rune(pathSpec[0]) == rune('p') && rune(pathSpec[1]) == rune('@')) ||
		(rune(pathSpec[1]) == rune('p') && rune(pathSpec[2]) == rune('@')) {
		return true
	}

	return false
}

func IsParallelPathSpec(pathSpec []byte) bool {
	if len(pathSpec) < 3 {
		return false
	}
	if (rune(pathSpec[0]) == rune('p') && rune(pathSpec[1]) == rune('@')) ||
		(rune(pathSpec[1]) == rune('p') && rune(pathSpec[2]) == rune('@')) {
		return true
	}

	return false
}
