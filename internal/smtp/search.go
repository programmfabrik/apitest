package smtp

import (
	"fmt"
	"regexp"
)

// SearchByHeader returns the list of all given ContentHavers that
// have at least one header matching the given regular expression.
//
// Note that the regex is performed for each header value individually,
// including for multi-value headers. The header value is first serialized
// by concatenating it after the header name, colon and space. It is not
// being encoded as if for transport (e.g. quoted-printable),
// but concatenated as-is.
func SearchByHeader[T ContentHaver](haystack []T, re *regexp.Regexp) []T {
	out := make([]T, 0, len(haystack))

	for _, c := range haystack {
		if anyHeaderMatches(c.Content().Headers(), re) {
			out = append(out, c)
		}
	}

	return out
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
