package compare

import (
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/stretchr/testify/require"
)

func TestRejectBareControl(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
		have string
	}{
		{
			name: "control without a target",
			want: `{"body": {":control": {"no_extra": true}}}`,
			have: `{"body": {"extra": true}}`,
		},
		{
			name: "misplaced array control silently allows extra rows",
			want: `{"body": {"rows": [{"id": 1, ":control": {"order_matters": false, "no_extra": true}}]}}`,
			have: `{"body": {"rows": [{"id": 1}, {"id": 2}]}}`,
		},
		{
			name: "control alongside an empty key",
			want: `{"body": {"": {}, ":control": {"no_extra": true}}}`,
			have: `{"body": {"": {}}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var want, have any
			require.NoError(t, jsutil.UnmarshalString(tt.want, &want))
			require.NoError(t, jsutil.UnmarshalString(tt.have, &have))
			_, err := JsonEqual(want, have, ComparisonContext{})
			require.ErrorContains(t, err, `":control" must have a non-empty target key`)
		})
	}
}

func TestArrayControlOnSiblingKey(t *testing.T) {
	var want any
	require.NoError(t, jsutil.UnmarshalString(`{
		"rows": [{"id": 1}, {"id": 2}],
		"rows:control": {"order_matters": false, "no_extra": true}
	}`, &want))
	for _, tt := range []struct {
		name  string
		have  string
		equal bool
	}{
		{"reordered rows", `{"rows": [{"id": 2, "extra_field": true}, {"id": 1}]}`, true},
		{"extra row", `{"rows": [{"id": 2}, {"id": 1}, {"id": 3}]}`, false},
		{"missing row", `{"rows": [{"id": 1}]}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var have any
			require.NoError(t, jsutil.UnmarshalString(tt.have, &have))
			result, err := JsonEqual(want, have, ComparisonContext{})
			require.NoError(t, err)
			require.Equal(t, tt.equal, result.Equal, result.Failures)
		})
	}
}
