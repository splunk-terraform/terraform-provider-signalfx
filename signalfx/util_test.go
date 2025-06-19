// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSortByAscending(t *testing.T) {
	_, errors := validateSortBy("+foo", "sort_by")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSortByDescending(t *testing.T) {
	_, errors := validateSortBy("-foo", "sort_by")
	assert.Equal(t, 0, len(errors))
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
