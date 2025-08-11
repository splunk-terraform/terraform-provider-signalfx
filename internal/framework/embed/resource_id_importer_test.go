// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestResourceIDImporter_ImportState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		req    resource.ImportStateRequest
		data   tftypes.Value
		expect diag.Diagnostics
	}{
		{
			name:   "no id value set",
			req:    resource.ImportStateRequest{},
			expect: nil,
		},
		{
			name: "request set id field",
			req: resource.ImportStateRequest{
				ID: "test-id",
			},
			data: tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"id": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"id": tftypes.NewValue(tftypes.String, "set"),
				},
			),
		},
		{
			name: "No data set",
			req: resource.ImportStateRequest{
				ID: "test-id",
			},
			data: tftypes.Value{},
			expect: diag.Diagnostics{
				diag.WithPath(path.Empty().AtName("id"), diag.NewErrorDiagnostic(
					"State Write Error",
					"An unexpected error was encountered trying to write an attribute to the state. This is always an error in the provider. Please report the following to the provider developer:\n\nError: Cannot transform data: invalid transform: value missing type")),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				importer = ResourceIDImporter{}
				resp     = &resource.ImportStateResponse{
					State: tfsdk.State{
						Schema: schema.Schema{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						Raw: tc.data,
					},
				}
			)

			importer.ImportState(context.TODO(), tc.req, resp)
			assert.Equal(t, tc.expect, resp.Diagnostics, "Must match expected diagnostics")
		})
	}
}
