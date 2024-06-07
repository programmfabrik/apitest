package util

import (
	"strconv"
	"strings"
)

// PathSpec is a path specifier for including tests within manifests.
type PathSpec struct {
	// ParallelRuns matches the number of parallel runs specified
	// in a path spec like "5@foo.json"
	ParallelRuns int

	// Path matches the literal path, e.g. foo.json in "@foo.json"
	Path string
}

// ParsePathSpec tries to parse the given string into a PathSpec.
//
// It returns a boolean result that indicates if parsing was successful (i.e. if
// s is a valid path specifier). The string takes the format "[n]@file.json".
func ParsePathSpec(s string) (spec PathSpec, ok bool) {
	var parallelRuns string
	parallelRuns, spec.Path, ok = strings.Cut(s, "@")
	if parallelRuns != "" {
		spec.ParallelRuns, _ = strconv.Atoi(parallelRuns)
	} else {
		spec.ParallelRuns = 1
	}
	if !ok || spec.Path == "" || spec.ParallelRuns <= 0 {
		return PathSpec{}, false
	}
	return spec, true
}

// IsPathSpec is a wrapper around ParsePathSpec that discards the parsed PathSpec.
// It's useful for chaining within boolean expressions.
func IsPathSpec(s string) bool {
	_, ok := ParsePathSpec(s)
	return ok
}
