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
					Repetitions: 1,
					Path:        "foo.json",
				},
			},
			{
				s: "5@bar.json",
				expected: PathSpec{
					Repetitions: 5,
					Path:        "bar.json",
				},
			},
			{
				s: "123@baz.json",
				expected: PathSpec{
					Repetitions: 123,
					Path:        "baz.json",
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

			t.Run(testCase.s+" quoted", func(t *testing.T) {
				s := `"` + testCase.s + `"`

				actual, ok := ParsePathSpec(s)
				require.True(t, ok)
				require.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("invalid path specs are detected", func(t *testing.T) {
		testCases := []string{
			"",                         // empty
			`"@foo.json`, `@foo.json"`, // superfluous quotes
			`foo@bar.baz`, `1.23@foo.json`, // non-digit repetitions
			`p@old.syntax`, `p5@old.syntax`, `p123@old.syntax`, // old syntax
		}

		for _, testCase := range testCases {
			s := testCase

			t.Run(s, func(t *testing.T) {
				actual, ok := ParsePathSpec(s)
				require.False(t, ok)
				require.Zero(t, actual)
			})

			t.Run(s+" quoted", func(t *testing.T) {
				sq := `"` + s + `"`

				actual, ok := ParsePathSpec(sq)
				require.False(t, ok)
				require.Zero(t, actual)
			})
		}
	})
}
