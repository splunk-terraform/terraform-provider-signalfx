// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestSeverityLevel(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    any
		expect diag.Diagnostics
	}{
		{
			name: "No value provided",
			val:  nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name:   "expected value",
			val:    "Major",
			expect: nil,
		},
		{
			name: "invalid value",
			val:  "Fatal",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "value \"Fatal\" is not allowed; must be one of: [Critical Major Minor Warning Info]",
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := SeverityLevel()(tc.val, cty.Path{})
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
