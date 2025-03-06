// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestFilterAction(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		val   any
		diags diag.Diagnostics
	}{
		{
			name: "no value provided",
			val:  nil,
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name: "not a valid filter type",
			val:  "nop",
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "value \"nop\" is not allowed; expected to be one of: [Exclude Include]"},
			},
		},
		{
			name:  "valid filter type",
			val:   "Include",
			diags: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diags := FilterAction()(tc.val, cty.Path{})
			assert.Equal(t, tc.diags, diags, "Must match the expected values")
		})
	}
}
