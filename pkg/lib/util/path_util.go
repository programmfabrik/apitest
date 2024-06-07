package util

import (
	"regexp"
	"strconv"
)

/*
throughout this file we assume 'manifestDir' to be an absolute path
*/

var pathSpecRegex *regexp.Regexp = regexp.MustCompile(`^([0-9]*)@([^"]+)$`)

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
// It returns a boolean result that indicates if parsing was successful
// (i.e. if s is a valid path specifier).
func ParsePathSpec(s string) (PathSpec, bool) {
	// Remove outer quotes, if present
	if len(s) >= 2 && s[0] == '"' {
		if s[len(s)-1] != '"' {
			// path spec must have matching quotes, if quotes are present
			return PathSpec{}, false
		}

		s = s[1 : len(s)-1]
	}

	// Parse
	matches := pathSpecRegex.FindStringSubmatch(s)
	if matches == nil {
		return PathSpec{}, false
	}

	spec := PathSpec{
		ParallelRuns: 1,
		Path:         matches[2], // can't be empty, or else it wouldn't match the regex
	}

	// Determine number of parallel runs, if supplied
	if matches[1] != "" {
		prs, err := strconv.Atoi(matches[1])
		if err != nil {
			// matches[1] is all digits, so there must be something seriously wrong
			panic("error Atoi-ing all-decimal regex match")
		}

		spec.ParallelRuns = prs
	}

	return spec, true
}

// IsPathSpec is a wrapper around ParsePathSpec that discards the parsed PathSpec.
// It's useful for chaining within boolean expressions.
func IsPathSpec(s string) bool {
	_, ok := ParsePathSpec(s)
	return ok
}
