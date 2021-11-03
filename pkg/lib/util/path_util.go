package util

import (
	"regexp"
	"strconv"
	"strings"
)

/*
throughout this file we assume 'manifestDir' to be an absolute path
*/

func IsPathSpec(pathSpec string) bool {
	if len(pathSpec) < 3 {
		return false
	}
	if strings.HasPrefix(pathSpec, "@") {
		return true
	}
	if strings.HasPrefix(pathSpec, "p@") {
		return true
	}

	return IsParallelPathSpec(pathSpec)
}

func IsParallelPathSpec(pathSpec string) bool {
	n, _ := GetParallelPathSpec(pathSpec)
	return n > 0
}

func GetParallelPathSpec(pathSpec string) (parallelRepititions int, parsedPath string) {
	regex := *regexp.MustCompile(`^p(\d+)@(.+)$`)
	res := regex.FindAllStringSubmatch(pathSpec, -1)

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
