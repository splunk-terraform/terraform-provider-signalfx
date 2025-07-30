// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestColorName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		val   any
		diags diag.Diagnostics
	}{
		{
			name: "no values provided",
			val:  nil,
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name: "not a valid color",
			val:  "nop",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary: "value \"nop\" is not allowed; must be one of " +
						"[gray blue azure navy brown orange yellow magenta red pink violet purple lilac emerald chartreuse yellowgreen gold iris green jade cerise aquamarine]",
				},
			},
		},
		{
			name:  "valid color",
			val:   "red",
			diags: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diags := ColorName()(tc.val, cty.Path{})
			assert.Equal(t, tc.diags, diags, "Must match the expected values")
		})
	}
}

func TestColorHexValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		val   any
		diags diag.Diagnostics
	}{
		{
			name: "no values provided",
			val:  nil,
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name: "not a valid hex color code",
			val:  "nop",
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "value \"nop\" is not a valid hex color code"},
			},
		},
		{
			name:  "valid hex color code",
			val:   "#123456",
			diags: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diags := ColorHexValue()(tc.val, cty.Path{})
			assert.Equal(t, tc.diags, diags, "Must match the expected values")
		})
	}
}
