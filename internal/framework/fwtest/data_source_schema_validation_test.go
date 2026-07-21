// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validatingDataSource struct {
	schema schema.Schema
}

func (validatingDataSource) Metadata(context.Context, datasource.MetadataRequest, *datasource.MetadataResponse) {
}

func (d validatingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = d.schema
}

func (validatingDataSource) Read(context.Context, datasource.ReadRequest, *datasource.ReadResponse) {}

type validatingDataSourceModel struct {
	Name types.String `tfsdk:"name"`
}

type validatingNestedDataSourceModel struct {
	Settings struct {
		Enabled types.Bool `tfsdk:"enabled"`
	} `tfsdk:"settings"`
}

func TestDataSourceSchemaValidate(t *testing.T) {
	t.Parallel()

	valid := validatingDataSource{schema: schema.Schema{
		Description: "Looks up an example.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Required: true, Description: "Name to find."},
		},
	}}
	require.NoError(t, DataSourceSchemaValidate(valid, validatingDataSourceModel{}))

	missingDescriptions := validatingDataSource{schema: schema.Schema{
		Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Required: true}},
	}}
	err := DataSourceSchemaValidate(missingDescriptions, validatingDataSourceModel{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing schema description")
	assert.ErrorContains(t, err, `field "name" has no description`)

	wrongModel := struct {
		Other types.String `tfsdk:"other"`
	}{}
	err = DataSourceSchemaValidate(valid, wrongModel)
	require.Error(t, err)
	assert.ErrorContains(t, err, `expected field not found in model: "name"`)
	assert.ErrorContains(t, err, `additional field defined in model but not defined: "other"`)

	t.Run("missing attribute definitions", func(t *testing.T) {
		t.Parallel()

		dataSource := validatingDataSource{schema: schema.Schema{Description: "Looks up an example."}}
		assert.EqualError(t, DataSourceSchemaValidate(dataSource, validatingDataSourceModel{}), "missing schema attribute definitions")
	})

	t.Run("invalid model", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, DataSourceSchemaValidate(valid, "not a struct"), "model must be a struct, provided: string")
	})

	t.Run("mismatched field type", func(t *testing.T) {
		t.Parallel()

		model := struct {
			Name types.Int64 `tfsdk:"name"`
		}{}
		err := DataSourceSchemaValidate(valid, model)
		assert.EqualError(t, err, `field "name" has type "basetypes.StringType", expected "basetypes.Int64Type"`)
	})

	t.Run("markdown descriptions and nested model", func(t *testing.T) {
		t.Parallel()

		dataSource := validatingDataSource{schema: schema.Schema{
			MarkdownDescription: "Looks up nested settings.",
			Attributes: map[string]schema.Attribute{
				"settings": schema.SingleNestedAttribute{
					MarkdownDescription: "Settings to use.",
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{Required: true, MarkdownDescription: "Whether it is enabled."},
					},
				},
			},
		}}
		assert.NoError(t, DataSourceSchemaValidate(dataSource, validatingNestedDataSourceModel{}))
	})
}
