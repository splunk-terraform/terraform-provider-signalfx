// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

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
		value      types.String
		coordinate bool
		want       string
		wantSet    bool
		wantError  string
	}{
		"null": {
			value: types.StringNull(),
		},
		"fixed number": {
			value:   types.StringValue("4"),
			want:    `4`,
			wantSet: true,
		},
		"fixed number with whitespace": {
			value:   types.StringValue(" 4 "),
			want:    `4`,
			wantSet: true,
		},
		"relative fraction": {
			value:   types.StringValue("6/12"),
			want:    `"6/12"`,
			wantSet: true,
		},
		"clamped length": {
			value:   types.StringValue(`{"value":"1/2","min":4,"max":"100%"}`),
			want:    `{"max":"100%","min":4,"value":"1/2"}`,
			wantSet: true,
		},
		"coordinate array": {
			value:      types.StringValue(`["1/4",8,{"value":2,"min":1}]`),
			coordinate: true,
			want:       `["1/4",8,{"min":1,"value":2}]`,
			wantSet:    true,
		},
		"malformed JSON": {
			value:     types.StringValue(`{"value":`),
			wantError: "must contain valid JSON",
		},
		"multiple JSON values": {
			value:     types.StringValue(`{"value":1} {"value":2}`),
			wantError: "exactly one JSON value",
		},
		"missing clamped value": {
			value:     types.StringValue(`{"min":1}`),
			wantError: "must contain value",
		},
		"unknown clamped field": {
			value:     types.StringValue(`{"value":1,"other":2}`),
			wantError: `do not support "other"`,
		},
		"nested clamped bound": {
			value:     types.StringValue(`{"value":1,"min":{"value":0}}`),
			wantError: "clamped min",
		},
		"array outside coordinate": {
			value:     types.StringValue(`[1,2]`),
			wantError: "only for x and y",
		},
		"invalid coordinate part": {
			value:      types.StringValue(`[1,true]`),
			coordinate: true,
			wantError:  "coordinate part 1",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, set, err := dashifyLayoutValue(test.value, test.coordinate)
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

func TestDashifyLayoutStringCanonicalizesAdvancedValues(t *testing.T) {
	t.Parallel()

	item := map[string]any{
		"w": map[string]any{"value": "1/2", "max": "100%", "min": float64(4)},
		"x": []any{"1/4", float64(8)},
	}
	width, err := dashifyLayoutString(item, "w", false)
	require.NoError(t, err)
	assert.JSONEq(t, `{"max":"100%","min":4,"value":"1/2"}`, width.ValueString())
	x, err := dashifyLayoutString(item, "x", true)
	require.NoError(t, err)
	assert.Equal(t, `["1/4",8]`, x.ValueString())
	assert.Empty(t, item)
}

func TestDashifyLayoutStringRejectsUnsupportedShapes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value      any
		coordinate bool
		want       string
	}{
		"array width":        {value: []any{float64(1)}, want: "only for x and y"},
		"missing value":      {value: map[string]any{"min": float64(1)}, want: "must contain value"},
		"invalid coordinate": {value: []any{false}, coordinate: true, want: "coordinate part 0"},
		"boolean":            {value: true, want: "rather than a layout length"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			item := map[string]any{"value": test.value}
			_, err := dashifyLayoutString(item, "value", test.coordinate)
			require.ErrorContains(t, err, test.want)
		})
	}
}
