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
			"gray":        0,
			"blue":        1,
			"azure":       2,
			"navy":        3,
			"brown":       4,
			"orange":      5,
			"yellow":      6,
			"magenta":     7,
			"red":         8,
			"pink":        9,
			"violet":      10,
			"purple":      11,
			"lilac":       12,
			"emerald":     13,
			"chartreuse":  14,
			"yellowgreen": 15,
			"gold":        16,
			"iris":        17,
			"green":       18,
			"jade":        19,
			"cerise":      20,
			"aquamarine":  21,
		},
		// These values should be exactly matching to:
		// https://dev.splunk.com/observability/docs/chartsdashboards/charts_overview/#Chart-color-palettes (Values may differ as values have been updated to match the UI values)
		index: []string{
			0:  "#999999",
			1:  "#0077c2",
			2:  "#00b9ff",
			3:  "#6ca2b7",
			4:  "#b04600",
			5:  "#f47e00",
			6:  "#e5b312",
			7:  "#bd468d",
			8:  "#e9008a",
			9:  "#ff8dd1",
			10: "#876ff3",
			11: "#a747ff",
			12: "#ab99bc",
			13: "#007c1d",
			14: "#05ce00",
			15: "#0dba8f",
			16: "#eac24b",
			17: "#e5e517",
			18: "#6bd37e",
			19: "#aecf7f",
			20: "#e9008a",
			21: "#98abbe",
		},
	}
}

func (cp ColorPalette) ColorIndex(name string) (int32, bool) {
	index, exist := cp.named[name]
	return index % 16, exist // Since accepted colors are 16. Making sure that the index returned is within 16
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
	names := make([]string, int(16))
	for name, idx := range cp.named {
		if idx < 16 {
			names[idx] = name
		}
	}
	return names
}
