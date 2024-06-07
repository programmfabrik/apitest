package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePathSpec(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		testCases := []struct {
			s        string
			expected PathSpec
		}{
			{
				s: "@foo.json",
				expected: PathSpec{
					ParallelRuns: 1,
					Path:         "foo.json",
				},
			},
			{
				s: "5@bar.json",
				expected: PathSpec{
					ParallelRuns: 5,
					Path:         "bar.json",
				},
			},
			{
				s: "123@baz.json",
				expected: PathSpec{
					ParallelRuns: 123,
					Path:         "baz.json",
				},
			},
		}

		for i := range testCases {
			testCase := testCases[i]

			t.Run(testCase.s, func(t *testing.T) {
				actual, ok := ParsePathSpec(testCase.s)
				require.True(t, ok)
				require.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("invalid path specs are detected", func(t *testing.T) {
		testCases := []string{
			"",                             // empty
			`foo@bar.baz`, `1.23@foo.json`, // non-digit parallel runs
			`p@old.syntax`, `p5@old.syntax`, `p123@old.syntax`, // old syntax
		}

		for _, testCase := range testCases {
			s := testCase

			t.Run(s, func(t *testing.T) {
				actual, ok := ParsePathSpec(s)
				require.False(t, ok)
				require.Zero(t, actual)
			})
		}
	})
}
