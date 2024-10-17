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
						"[dark_red red crayola peridot greenyellow lime_green sage " +
						"gray azure blue light_blue azue brown dark_orange orange dark_yellow " +
						"gold yellow grape magenta cerise pink violet purple indigo lilac " +
						"dark_green emerald jade chartreuse green aquamarine grey_blue iris navy]",
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
