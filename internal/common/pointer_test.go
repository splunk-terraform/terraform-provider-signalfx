// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsPointer(t *testing.T) {
	t.Parallel()

	val := 6
	assert.IsType(t, (*int)(nil), AsPointer(val), "Must match the expected type")
	assert.Equal(t, &val, AsPointer(val), "Must match the expected value")
}

func TestAsPointerOnCondition(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    int
		cond   func(int) bool
		expect *int
	}{
		{
			name: "condition true",
			val:  10,
			cond: func(i int) bool {
				return i > 0
			},
			expect: AsPointer(10),
		},
		{
			name:   "condition false",
			val:    7,
			cond:   func(i int) bool { return i%2 == 0 },
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, AsPointerOnCondition(tc.val, tc.cond))
		})
	}
}
