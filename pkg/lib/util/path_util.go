package util

import (
	"path/filepath"
	"regexp"
	"strconv"
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

	if rune(pathSpec[0]) == rune('@') {
		return true
	}
	if rune(pathSpec[0]) == rune('p') && rune(pathSpec[1]) == rune('@') {
		return true
	}

	return IsParallelPathSpec(pathSpec)
}

func IsParallelPathSpec(pathSpec []byte) bool {
	n, _ := GetParallelPathSpec(pathSpec)
	return n > 0
}

func GetParallelPathSpec(pathSpec []byte) (parallelRepititions int, parsedPath string) {
	regex := *regexp.MustCompile(`^p(\d+)@(.+)$`)
	res := regex.FindAllStringSubmatch(string(pathSpec), -1)

	if len(res) != 1 {
		return 0, ""
	}
	if len(res[0]) != 3 {
		return 0, ""
	}

	parsedPath = res[0][2]
	parallelRepititions, err := strconv.Atoi(res[0][1])
	if err != nil {
		return 0, parsedPath
	}

	return parallelRepititions, parsedPath
}
