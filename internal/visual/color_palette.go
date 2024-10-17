// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package visual

type ColorPalette struct {
	// Named is the convience lookup table that allows
	// a user to type the color they want to use and
	// not need to be aware of the palette.
	//
	// Note:
	// - the names used for the colors are best guesses since they are not named within the documentation.
	named map[string]int32
	// Index are the values table that is defined in
	// https://dev.splunk.com/observability/docs/chartsdashboards/charts_overview/#Chart-color-palettes
	index []string
}

func NewColorPalette() ColorPalette {
	return ColorPalette{
		named: map[string]int32{
			"red":         0,
			"gold":        1,
			"iris":        2,
			"green":       3,
			"jade":        4,
			"gray":        5,
			"blue":        6,
			"azure":       7,
			"navy":        8,
			"brown":       9,
			"orange":      10,
			"yellow":      11,
			"magenta":     12,
			"cerise":      13,
			"pink":        14,
			"violet":      15,
			"purple":      16,
			"lilac":       17,
			"emerald":     18,
			"chartreuse":  19,
			"yellowgreen": 20,
			"aquamarine":  21,
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

func (cp ColorPalette) ColorIndex(name string) (int32, bool) {
	index, exist := cp.named[name]
	return index, exist
}

func (cp ColorPalette) IndexColorName(index int32) (string, bool) {
	color := ""
	for name, idx := range cp.named {
		if index == idx {
			color = name
		}
	}
	return color, color != ""
}

func (cp ColorPalette) HexCodebyIndex(index int32) (string, bool) {
	hex := ""
	if int(index) < len(cp.index) {
		hex = cp.index[index]
	}
	return hex, hex != ""
}

func (cp ColorPalette) Names() []string {
	names := make([]string, len(cp.named))
	for name, idx := range cp.named {
		names[idx] = name
	}
	return names
}
