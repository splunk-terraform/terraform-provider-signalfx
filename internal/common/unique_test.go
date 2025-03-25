// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		lists  [][]string
		expect []string
	}{
		{
			name:   "no values",
			lists:  [][]string{},
			expect: nil,
		},
		{
			name: "single value",
			lists: [][]string{
				{"The", "Quick", "Brown", "Horse"},
			},
			expect: []string{"The", "Quick", "Brown", "Horse"},
		},
		{
			name: "mixed values",
			lists: [][]string{
				{"The", "Quick", "Brown", "Horse"},
				{"Jumps", "Over", "Brown", "Fence"},
				{"Brown", "Horse", "Quick", "Battery", "Stable"},
			},
			expect: []string{"The", "Quick", "Brown", "Horse", "Jumps", "Over", "Fence", "Battery", "Stable"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tc.expect,
				Unique(tc.lists...),
				"Must match the expected values",
			)
		})
	}
}
