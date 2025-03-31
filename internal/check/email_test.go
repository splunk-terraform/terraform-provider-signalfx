// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestEmail(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		in     any
		expect diag.Diagnostics
	}{
		{
			name: "nil",
			in:   nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name: "incomplete email",
			in:   "example",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "mail: missing '@' or angle-addr"},
			},
		},
		{
			name:   "complete email",
			in:     "user@example.com",
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, Email(tc.in, cty.Path{}))
		})
	}
}
