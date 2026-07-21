// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestExistingResourcesRejectInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name           string
		implementation resource.Resource
	}{
		{name: "BigPanda", implementation: &ResourceBigPanda{}},
		{name: "Splunk On-Call", implementation: &ResourceSplunkOncall{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			schemaResponse := &resource.SchemaResponse{}
			test.implementation.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
			invalidPlan := tfsdk.Plan{
				Raw:    tftypes.NewValue(tftypes.Bool, true),
				Schema: schemaResponse.Schema,
			}
			invalidState := tfsdk.State{
				Raw:    tftypes.NewValue(tftypes.Bool, true),
				Schema: schemaResponse.Schema,
			}

			createResponse := &resource.CreateResponse{}
			test.implementation.Create(ctx, resource.CreateRequest{Plan: invalidPlan}, createResponse)
			assert.True(t, createResponse.Diagnostics.HasError())

			readResponse := &resource.ReadResponse{}
			test.implementation.Read(ctx, resource.ReadRequest{State: invalidState}, readResponse)
			assert.True(t, readResponse.Diagnostics.HasError())

			updateResponse := &resource.UpdateResponse{}
			test.implementation.Update(ctx, resource.UpdateRequest{Plan: invalidPlan}, updateResponse)
			assert.True(t, updateResponse.Diagnostics.HasError())

			deleteResponse := &resource.DeleteResponse{}
			test.implementation.Delete(ctx, resource.DeleteRequest{State: invalidState}, deleteResponse)
			assert.True(t, deleteResponse.Diagnostics.HasError())
		})
	}
}
