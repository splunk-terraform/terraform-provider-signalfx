// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestAttributeTypeMap(t *testing.T) {
	t.Parallel()

	result := AttributeTypeMap(map[string]schema.StringAttribute{"name": {Optional: true}})
	assert.Len(t, result, 1)
	assert.Equal(t, types.StringType, result["name"])
}

func TestStringValueOrEmpty(t *testing.T) {
	t.Parallel()

	assert.True(t, StringValueOrEmpty("").IsNull())
	assert.Equal(t, "value", StringValueOrEmpty("value").ValueString())
}
