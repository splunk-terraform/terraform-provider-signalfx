// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservabilityTemplateResourceModelMapping(t *testing.T) {
	ctx := context.Background()
	imports, diags := types.ListValueFrom(ctx, types.StringType, []string{"/v2/template/chart"})
	require.False(t, diags.HasError())
	datasource := &observabilityTemplateDatasourceModel{
		Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
		ProgramText: types.StringValue("data('requests').publish()"),
		SLOID:       types.StringNull(),
	}
	model := observabilityTemplateModel{
		Title:       types.StringValue("Example"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        types.StringValue(`{"<Chart>":[]}`),
		Metadata: &observabilityTemplateMetadataModel{
			Imports:    imports,
			Datasource: datasource,
		},
	}

	write, diags := observabilityTemplateWrite(ctx, model)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, template.RecordType, write.Type)
	assert.Equal(t, "Example", write.Title)
	assert.JSONEq(t, `{"<Chart>":[]}`, string(write.Spec))
	assert.Equal(t, []string{"/v2/template/chart"}, write.Metadata.Imports)
	require.NotNil(t, write.Metadata.Datasource)
	assert.Equal(t, template.DatasourceTypeSplunkObservability, write.Metadata.Datasource.Type)

	root := template.RootElementChart
	record := &template.Template{
		ID:       "template-id",
		Title:    "Example from API",
		Spec:     json.RawMessage(`{"<Chart>":[{"future":true}]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/chart"}},
	}
	state, err := observabilityTemplateModelFromRecord(model, record)
	require.NoError(t, err)
	assert.Equal(t, "template-id", state.ID.ValueString())
	assert.Equal(t, "Example from API", state.Title.ValueString())
	assert.Equal(t, string(template.RootElementChart), state.RootElement.ValueString())
	assert.JSONEq(t, `{"<Chart>":[{"future":true}]}`, state.Spec.ValueString())
	require.NotNil(t, state.Metadata)
	assert.Equal(t, imports, state.Metadata.Imports)
	assert.Same(t, datasource, state.Metadata.Datasource)
}

func TestObservabilityTemplateModelFromRecordLeavesOptionalMetadataUnsetOnImport(t *testing.T) {
	root := template.RootElement("FutureRootElement")
	record := &template.Template{
		ID:       "template-id",
		Title:    "Imported template",
		Spec:     json.RawMessage(`{"<FutureRootElement>":[]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/child"}},
	}

	state, err := observabilityTemplateModelFromRecord(observabilityTemplateModel{}, record)
	require.NoError(t, err)
	assert.Equal(t, "FutureRootElement", state.RootElement.ValueString())
	assert.Nil(t, state.Metadata)
}

func TestObservabilityTemplateModelFromRecordPreservesUnsetDirectImports(t *testing.T) {
	root := template.RootElementChart
	prior := observabilityTemplateModel{
		Metadata: &observabilityTemplateMetadataModel{
			Imports: types.ListNull(types.StringType),
		},
	}
	record := &template.Template{
		ID:       "template-id",
		Title:    "Template with descendants",
		Spec:     json.RawMessage(`{"<Chart>":[]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/direct", "/v2/template/descendant"}},
	}

	state, err := observabilityTemplateModelFromRecord(prior, record)
	require.NoError(t, err)
	require.NotNil(t, state.Metadata)
	assert.True(t, state.Metadata.Imports.IsNull())
}

func TestObservabilityTemplateWriteWithoutOptionalMetadata(t *testing.T) {
	model := observabilityTemplateModel{
		Title:       types.StringValue("Example"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        types.StringValue(`{"<Chart>":[]}`),
	}

	write, diags := observabilityTemplateWrite(context.Background(), model)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, write.Metadata.RootElement)
	assert.Equal(t, template.RootElementChart, *write.Metadata.RootElement)
	assert.Empty(t, write.Metadata.Imports)
	assert.Nil(t, write.Metadata.Datasource)
}

func TestObservabilityJSONEqual(t *testing.T) {
	assert.True(t, observabilityJSONEqual(`{"a":1,"b":2}`, "{\"b\": 2, \"a\": 1}"))
	assert.False(t, observabilityJSONEqual(`{"a":1}`, `{"a":2}`))
}
