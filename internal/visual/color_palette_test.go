// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package visual

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorPalette(t *testing.T) {
	t.Parallel()

	var (
		cp   = NewColorPalette()
		seen = make([]int, 22)
	)

	for _, name := range cp.Names() {
		idx, exist := cp.GetColorIndex(name)
		require.True(t, exist, "Must return a valid result reading index")
		seen[int(idx)]++
		hex, exist := cp.GetHexCodebyIndex(idx)
		assert.NotEmpty(t, hex, "Must have returned the expected hex code")
		assert.True(t, exist, "Must have found the hex code value")
	}

	for idx := range seen {
		assert.GreaterOrEqual(t, seen[idx], 1, "Must have seen each value at least once")
	}
}

func TestHistoricalNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"gray",
		"blue",
		"azure",
		"navy",
		"brown",
		"orange",
		"yellow",
		"magenta",
		"purple",
		"pink",
		"violet",
		"lilac",
		"iris",
		"emerald",
		"green",
		"aquamarine",
		"red",
		"gold",
		"greenyellow",
		"chartreuse",
		"jade",
	} {
		_, ok := NewColorPalette().GetColorIndex(name)
		assert.True(t, ok, "Must have the %q set as an option", name)
	}
}
