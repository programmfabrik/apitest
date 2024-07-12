package smtp

import (
	"fmt"
	"regexp"
)

// SearchByHeader returns the list of all given ContentHavers that,
// for each of the given regular expressions, has at least one header
// matching it (different regexes can be matched by different headers or
// the same header).
//
// Note that in the context of this function, a regex is performed for each
// header value individually, including for multi-value headers. The header
// value is first serialized by concatenating it after the header name, colon
// and space. It is not being encoded as if for transport (e.g. quoted-
// printable), but concatenated as-is.
func SearchByHeader[T ContentHaver](haystack []T, rxs ...*regexp.Regexp) []T {
	out := make([]T, 0, len(haystack))

	for _, c := range haystack {
		if allRegexesMatchAnyHeader(c.Content().Headers(), rxs) {
			out = append(out, c)
		}
	}

	return out
}

func allRegexesMatchAnyHeader(headers map[string][]string, rxs []*regexp.Regexp) bool {
	for _, re := range rxs {
		if !anyHeaderMatches(headers, re) {
			return false
		}
	}

	return true
}

func anyHeaderMatches(headers map[string][]string, re *regexp.Regexp) bool {
	for k, vs := range headers {
		for _, v := range vs {
			header := fmt.Sprintf("%s: %s", k, v)
			if re.MatchString(header) {
				return true
			}
		}
	}

	return false
}
