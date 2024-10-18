// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestNopDecodeTerraform(t *testing.T) {
	t.Parallel()

	v, err := NopDecodeTerraform[int](&schema.ResourceData{})
	assert.IsType(t, (*int)(nil), v)
	assert.Nil(t, v, "Must returned a nil value")
	assert.NoError(t, err, "Must not return an error")
}

func TestNopEncodeTerraform(t *testing.T) {
	t.Parallel()

	err := NopEncodeTerraform[int](new(int), &schema.ResourceData{})
	assert.NoError(t, err, "Must not error")
}
