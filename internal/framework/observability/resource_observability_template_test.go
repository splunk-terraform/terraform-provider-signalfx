// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceObservabilityTemplateMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityTemplate()
	var metadataResponse resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadataResponse)
	assert.Equal(t, "signalfx_observability_template", metadataResponse.TypeName)
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	assert.True(t, schemaResponse.Schema.Attributes["title"].IsRequired())
	rootElement, ok := schemaResponse.Schema.Attributes["root_element"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, rootElement.IsRequired())
	require.Len(t, rootElement.PlanModifiers, 1)
	assert.True(t, schemaResponse.Schema.Attributes["spec"].IsRequired())
	metadataBlock, ok := schemaResponse.Schema.Blocks["metadata"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.Empty(t, metadataBlock.Validators)
	assert.NotContains(t, metadataBlock.Attributes, "root_element")
	assert.True(t, metadataBlock.Attributes["imports"].IsOptional())
	datasource, ok := metadataBlock.Blocks["datasource"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.True(t, datasource.Attributes["type"].IsOptional())
	assert.True(t, datasource.Attributes["program_text"].IsOptional())
	assert.True(t, datasource.Attributes["slo_id"].IsOptional())
	assert.NotContains(t, schemaResponse.Schema.Attributes, "template_contents")
	assert.NotContains(t, schemaResponse.Schema.Attributes, "imports")
}

func TestResourceObservabilityTemplateRootElementValidation(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityTemplate()
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	attribute, ok := schemaResponse.Schema.Attributes["root_element"].(schema.StringAttribute)
	require.True(t, ok)
	require.Len(t, attribute.Validators, 1)

	for name, test := range map[string]struct {
		value       string
		expectError bool
	}{
		"chart":                  {value: "Chart"},
		"dashboard":              {value: "Dashboard"},
		"arbitrary root element": {value: "FutureRootElement"},
		"empty":                  {value: "", expectError: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := validator.StringRequest{
				Path:           path.Root("root_element"),
				PathExpression: path.MatchRoot("root_element"),
				ConfigValue:    types.StringValue(test.value),
			}
			response := validator.StringResponse{}
			attribute.Validators[0].ValidateString(context.Background(), request, &response)
			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
		})
	}
}

func TestObservabilityTemplateDatasourceValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root         string
		datasource   observabilityTemplateDatasourceModel
		expectErrors int
	}{
		"chart datasource": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
				ProgramText: types.StringValue("data('requests').publish()"),
				SLOID:       types.StringNull(),
			},
		},
		"chart program without type": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringNull(),
				ProgramText: types.StringValue("data('requests').publish()"),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"dashboard type without program": {
			root: "dashboard",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
				ProgramText: types.StringNull(),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"SLO datasource": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservabilitySLO)),
				ProgramText: types.StringValue("data('service.level').publish()"),
				SLOID:       types.StringValue("example-slo-id"),
			},
		},
		"SLO datasource without ID": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservabilitySLO)),
				ProgramText: types.StringValue("data('service.level').publish()"),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"custom root follows extensible API validation": {
			root: "FutureRootElement",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringNull(),
				ProgramText: types.StringValue("future datasource contents"),
				SLOID:       types.StringNull(),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			response := resource.ValidateConfigResponse{}
			validateObservabilityTemplateDatasource(types.StringValue(test.root), test.datasource, &response)
			assert.Len(t, response.Diagnostics, test.expectErrors)
		})
	}
}

func TestObservabilityTemplateSpecValidation(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateObservabilityTemplateSpec(`{"<Chart>":[]}`))
	assert.Error(t, validateObservabilityTemplateSpec(`{"<Chart>":`))
	assert.Error(t, validateObservabilityTemplateSpec(`null`))
	assert.Error(t, validateObservabilityTemplateSpec(`[]`))
}

func TestResourceObservabilityTemplateUnitTest(t *testing.T) {
	store := newTemplateAPIStore()

	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.RequireAbove(tfversion.Version0_12_26),
			},
			ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
				t,
				store.handlers(),
				fwtest.WithMockResources(NewResourceObservabilityTemplate),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_template.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_template.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "title", "Request rate"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "root_element", "Chart"),
						testresource.TestCheckNoResourceAttr("signalfx_observability_template.test", "metadata"),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_template_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_template.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "title", "Request rate (updated)"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "root_element", "Chart"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "metadata.imports.0", "/v2/template/shared"),
					),
				},
			},
		},
	)
}
