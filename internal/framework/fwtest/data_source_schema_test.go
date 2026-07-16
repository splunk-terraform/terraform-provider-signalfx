// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
)

func TestWalkDataSourceSchema(t *testing.T) {
	t.Parallel()

	input := map[string]schema.Attribute{
		"name": schema.StringAttribute{},
		"items": schema.ListNestedAttribute{NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{"value": schema.StringAttribute{}},
		}},
		"settings": schema.SingleNestedAttribute{Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{},
		}},
	}

	actual := make(map[string]schema.Attribute)
	for path, attribute := range WalkDataSourceSchema(input) {
		actual[path.String()] = attribute
	}

	assert.Contains(t, actual, "name")
	assert.Contains(t, actual, "items")
	assert.Contains(t, actual, "items[0].value")
	assert.Contains(t, actual, "settings")
	assert.Contains(t, actual, "settings.enabled")
}
