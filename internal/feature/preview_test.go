// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultPreview(t *testing.T) {
	t.Parallel()

	p, err := NewPreview()
	require.NoError(t, err, "Must not error when creating a default preview")

	assert.False(t, p.Enabled(), "Must be disabled by default")
	assert.False(t, p.GlobalAvailable(), "Must be disabled by default")
	assert.Equal(t, "", p.Description(), "Must have no description by default")
	assert.Equal(t, "", p.Introduced(), "Must have no value set for introduced")

	p.SetEnabled(true)
	assert.True(t, p.Enabled(), "Must be enabled once set")
}
