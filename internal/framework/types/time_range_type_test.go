// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtypes

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestTimeRangeTypeString(t *testing.T) {
	t.Parallel()

	trp := TimeRangeType{}
	assert.Equal(t, "fwtypes.TimeRangeType", trp.String(), "Must match the expected string representation")
}

func TestTimeRangeTypeValueType(t *testing.T) {
	t.Parallel()

	trp := TimeRangeType{}
	assert.Equal(t, TimeRange{}, trp.ValueType(context.Background()), "Must match the expected value type")
}

func TestTimeRangeTypeEqual(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		match attr.Type
		equal bool
	}{
		{
			name:  "nil typed value",
			match: attr.Type(nil),
			equal: false,
		},
		{
			name:  "exact same type",
			match: TimeRangeType{},
			equal: true,
		},
		{
			name:  "different type",
			match: basetypes.StringType{},
			equal: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			trp := TimeRangeType{}
			assert.Equal(t, tc.equal, trp.Equal(tc.match), "Must match the expected equality result")
		})
	}
}

func TestTimeRangeTypeValueFromString(t *testing.T) {
	t.Parallel()

	var (
		trp = TimeRangeType{}
		in  = basetypes.NewStringValue("5m")
	)

	out, diags := trp.ValueFromString(context.Background(), in)

	assert.Equal(t, TimeRange{StringValue: in}, out, "Must match the expected value")
	assert.Empty(t, diags, "There must not be any diagnostics")
}

func TestTimeRangeTypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		in     tftypes.Value
		expect attr.Value
		errVal string
	}{
		{
			name:   "wrong value type provided",
			in:     tftypes.NewValue(tftypes.Bool, false),
			expect: nil,
			errVal: "can't unmarshal tftypes.Bool into *string, expected string",
		},
		{
			name: "unknown value provided",
			in:   tftypes.Value{},
			expect: TimeRange{
				StringValue: basetypes.NewStringNull(),
			},
			errVal: "",
		},
		{
			name: "expected string value",
			in:   tftypes.NewValue(tftypes.String, "5m"),
			expect: TimeRange{
				StringValue: basetypes.NewStringValue("5m"),
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			trp := TimeRangeType{}
			out, err := trp.ValueFromTerraform(context.Background(), tc.in)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
				assert.Nil(t, out, "The output value must be nil")
			} else {
				assert.NoError(t, err, "There must not be an error")
				assert.Equal(t, tc.expect, out, "Must match the expected value")
			}
		})
	}
}
