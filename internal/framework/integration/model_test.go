// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationModel(t *testing.T) {
	t.Parallel()

	model := integrationModel{ID: types.StringValue("original-id")}
	model.update("First name", true)
	assert.Equal(t, types.StringValue("original-id"), model.ID)
	assert.Equal(t, types.StringValue("First name"), model.Name)
	assert.Equal(t, types.BoolValue(true), model.Enabled)

	model.updateWithID("updated-id", "Second name", false)
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Second name"), model.Name)
	assert.Equal(t, types.BoolValue(false), model.Enabled)

	attributes := integrationAttributes()
	assert.ElementsMatch(t, []string{"id", "name", "enabled"}, mapKeys(attributes))
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
