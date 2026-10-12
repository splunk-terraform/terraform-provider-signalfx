// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashifyLayoutValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value     types.String
		want      string
		wantSet   bool
		wantError string
	}{
		"null":                         {value: types.StringNull()},
		"fixed number":                 {value: types.StringValue("4"), want: `4`, wantSet: true},
		"fixed number with whitespace": {value: types.StringValue(" 4 "), want: `4`, wantSet: true},
		"relative fraction":            {value: types.StringValue("6/12"), want: `"6/12"`, wantSet: true},
		"percentage":                   {value: types.StringValue("100%"), want: `"100%"`, wantSet: true},
		"clamped length":               {value: types.StringValue(`{"value":"1/2","min":4}`), wantError: "JSON objects and arrays are not supported"},
		"coordinate array":             {value: types.StringValue(`["1/4",8]`), wantError: "JSON objects and arrays are not supported"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, set, err := dashifyLayoutValue(test.value)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantSet, set)
			if !set {
				return
			}
			encoded, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(encoded))
		})
	}
}

func TestDashifyLayoutStringKeepsAdvancedValuesUnmodeled(t *testing.T) {
	t.Parallel()

	item := map[string]any{
		"w": map[string]any{"value": "1/2", "max": "100%"},
		"h": json.Number("20"),
	}
	width, err := dashifyLayoutString(item, "w")
	require.NoError(t, err)
	assert.True(t, width.IsNull())
	assert.Contains(t, item, "w")
	height, err := dashifyLayoutString(item, "h")
	require.NoError(t, err)
	assert.Equal(t, "20", height.ValueString())
	assert.NotContains(t, item, "h")
}
