// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestTimeRange(t *testing.T) {
	t.Parallel()

	// More thorough testing is done within common
	for _, tc := range []struct {
		name   string
		value  any
		expect diag.Diagnostics
	}{
		{
			name:  "no values provided",
			value: nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be type string"},
			},
		},
		{
			name:  "invalid timestamp",
			value: "last week",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "invalid timerange \"last week\": no negative prefix"},
			},
		},
		{
			name:   "valid timestamp",
			value:  "-1w",
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := TimeRange()(tc.value, cty.Path{})
			assert.Equal(t, tc.expect, actual, "Must match the expected values")
		})
	}
}
