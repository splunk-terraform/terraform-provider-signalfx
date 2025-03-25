// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOrderedSet(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewOrderedSet[int](), "Must have a valid set returned")
}

func TestOrderedSetAll(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		in     []int
		expect []int
	}{
		{
			name:   "No values",
			in:     []int{},
			expect: nil,
		},
		{
			name:   "unique values",
			in:     []int{0, 1, 2, 3, 4, 5},
			expect: []int{0, 1, 2, 3, 4, 5},
		},
		{
			name:   "duplicate values",
			in:     []int{0, 0, 0, 1},
			expect: []int{0, 1},
		},
		{
			name:   "unordered values",
			in:     []int{3, 3, 1, 1, 2, 2, 1, 0, 5, 5, 5, 1, 4},
			expect: []int{3, 1, 2, 0, 5, 4},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			os := NewOrderedSet[int]()
			os.Append(tc.in...)
			assert.Equal(t, tc.expect, slices.Collect(os.All()), "Must match the expected values")
		})
	}
}
