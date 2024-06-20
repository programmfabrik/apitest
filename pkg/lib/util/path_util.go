package util

import (
	"fmt"
	"io"
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
// The string takes the format "[n]@file.json". Invalid path specs
// result in an error.
func ParsePathSpec(s string) (*PathSpec, error) {
	var (
		ok           bool
		err          error
		parallelRuns string
		spec         PathSpec
	)

	parallelRuns, spec.Path, ok = strings.Cut(s, "@")
	if parallelRuns != "" {
		spec.ParallelRuns, err = strconv.Atoi(parallelRuns)
		if err != nil {
			return nil, fmt.Errorf("error parsing ParallelRuns of path spec %q: %w", s, err)
		}
	} else {
		spec.ParallelRuns = 1
	}

	if !ok || spec.Path == "" || spec.ParallelRuns < 0 {
		return nil, fmt.Errorf("invalid path spec %q", s)
	}

	return &spec, err
}

// IsPathSpec is a wrapper around ParsePathSpec that discards the parsed PathSpec.
// It's useful for chaining within boolean expressions.
func IsPathSpec(s string) bool {
	_, err := ParsePathSpec(s)
	return err == nil
}

// Load loads the contents of the file pointed to by the PathSpec into a byte array.
func (ps PathSpec) LoadContents(manifestDir string) ([]byte, error) {
	requestFile, err := OpenFileOrUrl(ps.Path, manifestDir)
	if err != nil {
		return nil, fmt.Errorf("error opening path: %w", err)
	}
	defer requestFile.Close()

	contents, err := io.ReadAll(requestFile)
	if err != nil {
		return nil, fmt.Errorf("error loading file at %q: %w", ps, err)
	}

	return contents, nil
}
