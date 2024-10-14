// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		path   string
		panics bool
	}{
		{
			name:   "not existing file",
			path:   "does-not-exist",
			panics: true,
		},
		{
			name:   "file exists",
			path:   "config_test.go",
			panics: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.panics {
				assert.Panics(t, func() {
					_ = LoadConfig(tc.path)
				})
			} else {
				assert.NotPanics(t, func() {
					_ = LoadConfig(tc.path)
				})
			}
		})
	}
}
