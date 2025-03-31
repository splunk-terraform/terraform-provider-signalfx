// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"testing"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultPreview(t *testing.T) {
	t.Parallel()

	p, err := NewPreview()
	require.NoError(t, err, "Must not error when creating a default preview")

	assert.False(t, p.Enabled(), "Must be disabled by default")
	assert.False(t, p.GlobalAvailable(), "Must be disabled by default")
	assert.Empty(t, p.Description(), "Must have no description by default")
	assert.Empty(t, p.Introduced(), "Must have no value set for introduced")

	p.SetEnabled(true)
	assert.True(t, p.Enabled(), "Must be enabled once set")
}

func TestPreviewLogFields(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		preview *Preview
		expect  tfext.LogFields
	}{
		{
			name:    "no preview set",
			preview: nil,
			expect: tfext.LogFields{
				"feature.name": "feature.example-01",
				"error":        "feature preview is unset",
			},
		},
		{
			name: "preview set",
			preview: func(tb testing.TB) *Preview {
				p, err := NewPreview()
				assert.NoError(t, err, "Must not error creating preview")
				return p
			}(t),
			expect: tfext.LogFields{
				"feature.name":    "feature.example-01",
				"feature.ga":      false,
				"feature.enabled": false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tc.expect,
				NewPreviewLogFields("feature.example-01", tc.preview),
				"Must match the expected values",
			)
		})
	}
}
