// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package visual

import "slices"

type ColorPalette struct {
	// Named is the convience lookup table that allows
	// a user to type the color they want to use and
	// not need to be aware of the palette.
	//
	// Note:
	// - the names used for the colors are best guesses since they are not named within the documentation.
	// - Values can be referenced more than once to improve UX.
	named map[string]int32
	// Index are the values table that is defined in
	// https://dev.splunk.com/observability/docs/chartsdashboards/charts_overview/#Chart-color-palettes
	index []string
}

func NewColorPalette() ColorPalette {
	return ColorPalette{
		named: map[string]int32{
			"red":         0,
			"orange":      10,
			"yellow":      11,
			"green":       19,
			"blue":        6,
			"indigo":      17,
			"violet":      15,
			"brown":       9,
			"dark_red":    0,
			"crayola":     1,
			"dark_orange": 9,
			"peridot":     2,
			"dark_yellow": 11,
			"gold":        11,
			"lime_green":  3,
			"sage":        4,
			"dark_green":  18,
			"emerald":     18,
			"chartreuse":  19,
			"aquamarine":  20,
			"light_blue":  7,
			"navy":        21,
			"azue":        8,
			"iris":        21,
			"purple":      16,
			"magenta":     12,
			"grape":       12,
			"lilac":       17,
			"cerise":      13,
			"pink":        14,
			"gray":        5,
			"grey_blue":   21,
			"azure":       6,
			"greenyellow": 3,
			"jade":        18,
		},
		// These values should be exactly matching to:
		// https://dev.splunk.com/observability/docs/chartsdashboards/charts_overview/#Chart-color-palettes
		index: []string{
			0:  "#ea1849",
			1:  "#eac24b",
			2:  "#e5e517",
			3:  "#6bd37e",
			4:  "#aecf7f",
			5:  "#999999",
			6:  "#0077c2",
			7:  "#00b9ff",
			8:  "#6ca2b7",
			9:  "#b04600",
			10: "#f47e00",
			11: "#e5b312",
			12: "#bd468d",
			13: "#e9008a",
			14: "#ff8dd1",
			15: "#876ff3",
			16: "#a747ff",
			17: "#ab99bc",
			18: "#007c1d",
			19: "#05ce00",
			20: "#0dba8f",
			21: "#98abbe",
		},
	}
}

func (cp ColorPalette) GetColorIndex(name string) (int32, bool) {
	index, exist := cp.named[name]
	return index, exist
}

func (cp ColorPalette) GetHexCodebyIndex(index int32) (string, bool) {
	hex := ""
	if int(index) < len(cp.index) {
		hex = cp.index[index]
	}
	return hex, hex != ""
}

func (cp ColorPalette) Names() []string {
	words := make(map[int32][]string, len(cp.named))
	for name, index := range cp.named {
		words[index] = append(words[index], name)
	}
	names := make([]string, 0, len(cp.named))
	for i := int32(0); int(i) < len(cp.index); i++ {
		colors := words[i]
		slices.Sort(colors)
		names = append(names, colors...)
	}
	return names
}
