// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNameFromChartColorsByIndex(t *testing.T) {
	name, err := getNameFromChartColorsByIndex(4)
	assert.Equal(t, "dark_orange", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromChartColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestGetNameFromPaletteColorsByIndex(t *testing.T) {
	name, err := getNameFromPaletteColorsByIndex(2)
	assert.Equal(t, "azure", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromPaletteColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestGetNameFromFullPaletteColorsByIndex(t *testing.T) {
	name, err := getNameFromFullPaletteColorsByIndex(16)
	assert.Equal(t, "red", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromPaletteColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestValidateSortByAscending(t *testing.T) {
	_, errors := validateSortBy("+foo", "sort_by")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSortByDescending(t *testing.T) {
	_, errors := validateSortBy("-foo", "sort_by")
	assert.Equal(t, 0, len(errors))
}

func TestValidateFullPaletteColors(t *testing.T) {
	_, errors := validateFullPaletteColors("chartreuse", "color_theme")
	assert.Equal(t, 0, len(errors))
}

func TestValidateFullPaletteColorsFail(t *testing.T) {
	_, errors := validateFullPaletteColors("color_palette", "color_theme")
	assert.Equal(t, 1, len(errors))
}

func TestValidateSortByNoDirection(t *testing.T) {
	_, errors := validateSortBy("foo", "sort_by")
	assert.Equal(t, 1, len(errors))
}

func TestBuildAppURL(t *testing.T) {
	u, error := buildAppURL("https://www.example.com", "/chart/abc123")
	assert.NoError(t, error)
	assert.Equal(t, "https://www.example.com/#/chart/abc123", u)
}

func TestFlattenStringSliceToSet(t *testing.T) {
	set := flattenStringSliceToSet([]string{"a", "b"})
	assert.Equal(t, 2, set.Len(), "Set missing arguments")

	setWithEmptyStrings := flattenStringSliceToSet([]string{"a", "", "b"})
	assert.Equal(t, 2, setWithEmptyStrings.Len(), "Set missing arguments")
}
