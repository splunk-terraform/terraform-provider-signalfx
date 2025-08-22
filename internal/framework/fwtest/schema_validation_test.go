// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

type ResourceMock struct {
	resource.Resource

	schema schema.Schema
}

func (rm ResourceMock) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rm.schema
}

func TestResourceSchemaValidate(t *testing.T) {
	t.Parallel()

	id := schema.StringAttribute{
		Required:    true,
		Description: "The unique identifier for the resource.",
	}

	for _, tc := range []struct {
		name   string
		res    ResourceMock
		model  any
		expect string
	}{
		{
			name: "Invalid model type provided",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model:  "not a struct",
			expect: "model must be a struct, provided: string",
		},
		{
			name: "Missing schema definition within model",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model:  struct{}{},
			expect: `expected field not found in model: "id", check struct tags`,
		},
		{
			name: "Valid schema and model",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model: struct {
				Id types.String `tfsdk:"id"`
			}{},
		},
		{
			name: "Additional values defined in model not present in schema",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model: struct {
				Id       types.String `tfsdk:"id"`
				fastmode types.Bool   `tfsdk:"fastmode"`
			}{},
			expect: `framework requires field exported "fastmode"`,
		},
		{
			name: "Missed defining description in schema with missing struct tag",
			res: ResourceMock{
				schema: schema.Schema{
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model:  struct{ id types.String }{},
			expect: `missing schema description; expected field not found in model: "id", check struct tags`,
		},
		{
			name: "Missed defining attributes in schema",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes:  nil,
				},
			},
			model:  &struct{ id types.String }{},
			expect: "missing schema attribute definitions",
		},
		{
			name: "Model define additional fields not defined in schema",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model: struct {
				Id    types.String `tfsdk:"id"`
				Extra types.String `tfsdk:"extra"`
			}{},
			expect: `additional field defined in model but not defined: "extra"`,
		},
		{
			name: "Mismatched field type in schema and model",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "Test schema",
					Attributes: map[string]schema.Attribute{
						"id": id,
					},
				},
			},
			model: &struct {
				Id types.Int64 `tfsdk:"id"`
			}{},
			expect: `field "id" has type "basetypes.StringType", expected "basetypes.Int64Type"`,
		},
		{
			name: "no description applied",
			res: ResourceMock{
				schema: schema.Schema{
					Description: "awesome",
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			model: &struct {
				Id types.String `tfsdk:"id"`
			}{},
			expect: `field "id" has no description`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ResourceSchemaValidate(tc.res, tc.model)
			if tc.expect == "" {
				assert.NoError(t, err, "Must not return an error")
			} else {
				assert.EqualError(t, err, tc.expect, "Must return the expected error")
			}
		})
	}
}
