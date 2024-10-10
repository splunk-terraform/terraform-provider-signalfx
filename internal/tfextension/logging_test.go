// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogFields(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		lf     LogFields
		expect map[string]any
	}{
		{
			name:   "empty",
			lf:     NewLogFields(),
			expect: map[string]any{},
		},
		{
			name:   "contains error message",
			lf:     ErrorLogFields(errors.New("failed operation")),
			expect: map[string]any{"error": "failed operation"},
		},
		{
			name:   "no error message",
			lf:     ErrorLogFields(nil),
			expect: map[string]any{},
		},
		{
			name:   "duration set",
			lf:     NewLogFields().Duration("min-delay", 1*time.Minute+30*time.Second),
			expect: map[string]any{"min-delay": "1m30s"},
		},
		{
			name:   "custom field",
			lf:     NewLogFields().Field("example", 1),
			expect: map[string]any{"example": 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.EqualValues(t, tc.expect, tc.lf, "Must match the expected field")
		})
	}
}
