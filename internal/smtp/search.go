package smtp

import (
	"fmt"
	"regexp"
)

func searchByHeaderCommon(headerIdxList []map[string][]string, re *regexp.Regexp) []int {
	result := make([]int, 0, len(headerIdxList))

	for idx, headers := range headerIdxList {
		if anyHeaderMatches(headers, re) {
			result = append(result, idx)
		}
	}

	return result
}

func anyHeaderMatches(headers map[string][]string, re *regexp.Regexp) bool {
	for k, v := range headers {
		header := fmt.Sprintf("%s: %s", k, v)
		if re.MatchString(header) {
			return true
		}
	}

	return false
}
