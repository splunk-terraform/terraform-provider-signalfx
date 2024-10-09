// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestValueUnit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		value    any
		expected diag.Diagnostics
	}{
		{
			name:  "no value provided",
			value: nil,
			expected: diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "expected <nil> to be type string",
				},
			},
		},
		{
			name:     "Unit set",
			value:    "Byte",
			expected: nil,
		},
		{
			name:  "Invalid unit",
			value: "Tuesday",
			expected: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "expected \"Tuesday\" to be one of [Bit Kilobit Megabit Gigabit Terabit Petabit Exabit Zettabit Yottabit Byte Kibibyte Mebibyte Gibibyte Tebibyte Pebibyte Exbibyte Zebibyte Yobibyte Nanosecond Microsecond Millisecond Second Minute Hour Day Week]",
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diag := ValueUnit()(tc.value, cty.Path{})
			assert.Equal(t, tc.expected, diag, "Must match the expected value")
		})
	}
}
