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
		seen = make([]int, len(cp.Names()))
	)

	for _, name := range cp.Names() {
		idx, exist := cp.ColorIndex(name)
		require.True(t, exist, "Must return a valid result reading index")
		seen[int(idx)]++
		hex, exist := cp.HexCodebyIndex(idx)
		assert.NotEmpty(t, hex, "Must have returned the expected hex code")
		assert.True(t, exist, "Must have found the hex code value")
	}

	for idx := range seen {
		assert.Equal(t, 1, seen[idx], "Must have seen index %d once", idx)
	}
}

func TestColorPaletteIndexColorName(t *testing.T) {
	t.Parallel()

	cp := NewColorPalette()
	for idx, expect := range cp.Names() {
		//nolint:gosec // Ignore warning for int overflow
		actual, exist := cp.IndexColorName(int32(idx))
		assert.True(t, exist, "Color must exist")
		assert.Equal(t, expect, actual, "Must match the expect index %d", idx)
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
		"red",
		"pink",
		"violet",
		"purple",
		"lilac",
		"emerald",
		"chartreuse",
		"yellowgreen",
		"gold",
		"iris",
		"green",
		"jade",
		"cerise",
		"aquamarine",
	} {
		_, ok := NewColorPalette().ColorIndex(name)
		assert.True(t, ok, "Must have the %q set as an option", name)
	}
}
