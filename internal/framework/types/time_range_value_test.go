// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtypes

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestTimeRangeType(t *testing.T) {
	t.Parallel()

	tr := TimeRange{}
	assert.Equal(t, TimeRangeType{}, tr.Type(context.Background()), "Must match the expected type")
}

func TestTimeRangeEqual(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		val   attr.Value
		equal bool
	}{
		{
			name:  "nil typed value",
			val:   attr.Value(nil),
			equal: false,
		},
		{
			name:  "unmatched type",
			val:   basetypes.StringValue{},
			equal: false,
		},
		{
			name:  "same value",
			val:   TimeRange{},
			equal: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := TimeRange{}
			assert.Equal(t, tc.equal, tr.Equal(tc.val), "Must match the expected equality result")
		})
	}
}

func TestTimeRangeValidateAttribute(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    string
		expect diag.Diagnostics
	}{
		{
			name: "unknown value",
			val:  "",
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test_case"),
					"Invalid Time Range",
					"invalid timerange: empty string",
				),
			},
		},
		{
			name: "not a time range",
			val:  "abc",
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test_case"),
					"Invalid Time Range",
					"invalid timerange: unexpected character: 'a'",
				),
			},
		},
		{
			name:   "valid time range",
			val:    "10w5d2h30m15s",
			expect: nil,
		},
		{
			name:   "negative valid time range",
			val:    "-10w5d2h30m15s",
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				req = xattr.ValidateAttributeRequest{
					Path: path.Root("test_case"),
				}

				resp xattr.ValidateAttributeResponse
			)
			tr := TimeRange{
				StringValue: basetypes.NewStringValue(tc.val),
			}
			tr.ValidateAttribute(context.Background(), req, &resp)
			assert.Equal(t, tc.expect, resp.Diagnostics, "Must match the expected diagnostics")
		})
	}
}

func TestTimeRangeValidateParamter(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    string
		expect *function.FuncError
	}{
		{
			name: "unknown value",
			val:  "",
			expect: function.NewArgumentFuncError(
				1,
				"invalid timerange: empty string",
			),
		},
		{
			name: "not a time range",
			val:  "abc",
			expect: function.NewArgumentFuncError(
				1,
				"invalid timerange: unexpected character: 'a'",
			),
		},
		{
			name:   "valid time range",
			val:    "10w5d2h30m15s",
			expect: nil,
		},
		{
			name:   "negative valid time range",
			val:    "-10w5d2h30m15s",
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				req = function.ValidateParameterRequest{
					Position: 1,
				}
				resp function.ValidateParameterResponse
			)
			tr := TimeRange{
				StringValue: basetypes.NewStringValue(tc.val),
			}
			tr.ValidateParameter(context.Background(), req, &resp)
			assert.Equal(t, tc.expect, resp.Error, "Must match the expected error")
		})
	}
}

func TestTimeRangeParseDuration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    string
		expect time.Duration
		errVal string
	}{
		{
			name:   "valid time range",
			val:    "10w5d2h30m15s",
			expect: 10*7*24*time.Hour + 5*24*time.Hour + 2*time.Hour + 30*time.Minute + 15*time.Second,
			errVal: "",
		},
		{
			name:   "negative valid time range",
			val:    "-10w5d2h30m15s",
			expect: -(10*7*24*time.Hour + 5*24*time.Hour + 2*time.Hour + 30*time.Minute + 15*time.Second),
			errVal: "",
		},
		{
			name:   "missing units",
			val:    "10",
			expect: 0,
			errVal: "invalid timerange: expected unit [s m h d w]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := TimeRange{
				StringValue: basetypes.NewStringValue(tc.val),
			}
			actual, err := tr.ParseDuration()
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
				assert.Equal(t, time.Duration(0), actual, "The duration must be zero")
			} else {
				assert.NoError(t, err, "There must not be an error")
				assert.Equal(t, tc.expect, actual, "Must match the expected duration")
			}
		})
	}
}

func TestTimeRangeStringSemanticEquals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		other  basetypes.StringValuable
		equal  bool
		issues diag.Diagnostics
	}{
		{
			name:  "nil other value",
			other: basetypes.NewStringNull(),
			equal: false,
			issues: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Semantic Equality Check Error",
					"An expected value type was received while comparing semantic values",
				),
			},
		},
		{
			name: "unequal values",
			other: TimeRange{
				StringValue: basetypes.NewStringValue("10m"),
			},
			issues: nil,
			equal:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := TimeRange{
				StringValue: basetypes.NewStringValue("5m"),
			}
			equal, issues := tr.StringSemanticEquals(context.Background(), tc.other)
			assert.Equal(t, tc.equal, equal, "Must match the expected equality result")
			assert.Equal(t, tc.issues, issues, "Must match the expected diagnostics")
		})
	}
}
