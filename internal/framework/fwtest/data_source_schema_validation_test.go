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
}
