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
		actual, exist := cp.IndexColorName(int32(idx))
		assert.True(t, exist, "Color must exist")
		assert.Equal(t, expect, actual, "Must match the expect index %d", idx)
	}
}

func TestHistoricalNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"red",
		"gold",
		"iris",
		"jade",
		"gray",
		"azure",
		"blue",
		"navy",
		"brown",
		"orange",
		"yellow",
		"magenta",
		"purple",
		"pink",
		"violet",
		"lilac",
		"emerald",
		"green",
		"aquamarine",
		"yellowgreen",
		"chartreuse",
	} {
		_, ok := NewColorPalette().ColorIndex(name)
		assert.True(t, ok, "Must have the %q set as an option", name)
	}
}
