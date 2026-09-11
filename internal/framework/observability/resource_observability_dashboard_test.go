// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"encoding/json"
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

func TestResourceObservabilityDashboardMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityDashboard()
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadata)
	assert.Equal(t, "signalfx_observability_dashboard", metadata.TypeName)
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	require.False(t, schemaResponse.Schema.ValidateImplementation(context.Background()).HasError())
	assert.NotEmpty(t, schemaResponse.Schema.Description)
	controlBar, ok := schemaResponse.Schema.Blocks["control_bar"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.Contains(t, controlBar.Blocks, "time_range")
	assert.Contains(t, controlBar.Blocks, "density")
	assert.Contains(t, controlBar.Blocks, "pinned_filter")
	assert.Contains(t, controlBar.Blocks, "filter_set")
	rootLayout, ok := schemaResponse.Schema.Blocks["layout"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.Contains(t, rootLayout.Attributes, "gap")
	assert.Contains(t, rootLayout.Attributes, "step")
	defaults, ok := rootLayout.Blocks["defaults"].(schema.SingleNestedBlock)
	require.True(t, ok)
	for _, name := range []string{"absolute", "width", "height", "min_width", "max_width", "min_height", "max_height"} {
		assert.Contains(t, defaults.Attributes, name)
	}
	assert.NotContains(t, defaults.Attributes, "x")
	assert.NotContains(t, defaults.Attributes, "y")

	rootContainer, ok := schemaResponse.Schema.Blocks["container"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Contains(t, rootContainer.NestedObject.Blocks, "template")
	assert.Contains(t, rootContainer.NestedObject.Blocks, "section")
	assert.Contains(t, rootContainer.NestedObject.Blocks, "group")
	itemLayout, ok := rootContainer.NestedObject.Blocks["layout"].(schema.SingleNestedBlock)
	require.True(t, ok)
	for _, name := range []string{"absolute", "width", "height", "min_width", "max_width", "min_height", "max_height", "x", "y"} {
		assert.Contains(t, itemLayout.Attributes, name)
	}

	section, ok := rootContainer.NestedObject.Blocks["section"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.Contains(t, section.Attributes, "collapse")
	assert.Contains(t, section.Attributes, "collapsible")
	assert.Contains(t, section.Blocks, "layout")
	sectionContainer, ok := section.Blocks["container"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Contains(t, sectionContainer.NestedObject.Blocks, "template")
	assert.Contains(t, sectionContainer.NestedObject.Blocks, "group")
	assert.NotContains(t, sectionContainer.NestedObject.Blocks, "section")

	group, ok := rootContainer.NestedObject.Blocks["group"].(schema.SingleNestedBlock)
	require.True(t, ok)
	assert.Contains(t, group.Attributes, "headerless")
	assert.Contains(t, group.Blocks, "layout")
	groupContainer, ok := group.Blocks["container"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Contains(t, groupContainer.NestedObject.Blocks, "template")
	assert.NotContains(t, groupContainer.NestedObject.Blocks, "section")
	assert.NotContains(t, groupContainer.NestedObject.Blocks, "group")
}

func TestResourceObservabilityDashboardCompleteLayoutConfig(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
			t,
			nil,
			fwtest.WithMockResources(NewResourceObservabilityDashboard, NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{{
			ConfigFile:         config.StaticFile("testdata/observability_dashboard_layout.tf"),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestObservabilityLayoutOptionValidators(t *testing.T) {
	t.Parallel()

	block := observabilityLayoutOptionsBlock()
	for name, test := range map[string]struct {
		attribute   string
		value       float64
		expectError bool
	}{
		"zero gap":       {attribute: "gap", value: 0},
		"negative gap":   {attribute: "gap", value: -1, expectError: true},
		"one pixel step": {attribute: "step", value: 1},
		"zero step":      {attribute: "step", value: 0, expectError: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attribute, ok := block.Attributes[test.attribute].(schema.Float64Attribute)
			require.True(t, ok)
			require.Len(t, attribute.Validators, 1)
			request := validator.Float64Request{
				Path:           path.Root(test.attribute),
				PathExpression: path.MatchRoot(test.attribute),
				ConfigValue:    types.Float64Value(test.value),
			}
			response := validator.Float64Response{}
			attribute.Validators[0].ValidateFloat64(context.Background(), request, &response)
			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
		})
	}
}

func TestObservabilityContainerContentErrorsDescribeOneContainer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		level observabilityContainerLevel
		want  string
	}{
		"dashboard": {
			level: observabilityDashboardContainerLevel,
			want:  "each dashboard container must set exactly one content block: template, section, or group",
		},
		"section": {
			level: observabilitySectionContainerLevel,
			want:  "each container within a section must set exactly one content block: template or group",
		},
		"group": {
			level: observabilityGroupContainerLevel,
			want:  "each container within a group must set exactly one template block",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, observabilityContainerContentError(test.level))
		})
	}
}

func TestValidateObservabilityLayoutBlocks(t *testing.T) {
	t.Parallel()

	t.Run("empty item layout", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateObservabilityLayout(&resp, path.Root("layout"), &observabilityLayoutModel{})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "at least one")
	})

	t.Run("malformed item value", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateObservabilityLayout(&resp, path.Root("layout"), &observabilityLayoutModel{Width: types.StringValue(`{"min":1}`)})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "must contain value")
	})

	t.Run("coordinate array", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateObservabilityLayout(&resp, path.Root("layout"), &observabilityLayoutModel{X: types.StringValue(`["1/4",8]`)})
		assert.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	})

	t.Run("empty layout options", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateObservabilityLayoutOptions(&resp, path.Root("layout"), &observabilityLayoutOptionsModel{})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "gap, step, defaults")
	})

	t.Run("empty defaults", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateObservabilityLayoutOptions(&resp, path.Root("layout"), &observabilityLayoutOptionsModel{Defaults: &observabilityLayoutDefaultsModel{}})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "defaults must set")
	})
}

func TestObservabilityDashboardSpecRoundTrip(t *testing.T) {
	t.Parallel()

	model := observabilityDashboardModel{
		Title: types.StringValue("Service overview"),
		Layout: &observabilityLayoutOptionsModel{
			Gap:  types.Float64Value(0),
			Step: types.Float64Value(1),
			Defaults: &observabilityLayoutDefaultsModel{
				Absolute:  types.BoolValue(false),
				MinWidth:  types.StringValue("4"),
				MinHeight: types.StringValue("2"),
			},
		},
		Container: []observabilityDashboardContainerModel{
			{
				Layout: &observabilityLayoutModel{
					Absolute:  types.BoolValue(true),
					Width:     types.StringValue(`{"max":"100%","min":4,"value":"1/2"}`),
					Height:    types.StringValue("20"),
					MinWidth:  types.StringValue("4"),
					MaxWidth:  types.StringValue("100%"),
					MinHeight: types.StringValue("2"),
					MaxHeight: types.StringValue("30"),
					X:         types.StringValue(`["1/4",8]`),
					Y:         types.StringValue("12"),
				},
				Template: &observabilityTemplateReferenceModel{TemplateID: types.StringValue("chart-a")},
			},
			{
				Section: &observabilitySectionModel{
					Title:       types.StringValue("Latency"),
					Collapse:    types.BoolValue(false),
					Collapsible: types.BoolValue(true),
					Layout: &observabilityLayoutOptionsModel{
						Gap:  types.Float64Value(8),
						Step: types.Float64Value(8),
					},
					Container: []observabilitySectionContainerModel{
						{
							Group: &observabilityGroupModel{
								Title:      types.StringValue("By service"),
								Headerless: types.BoolValue(true),
								Layout: &observabilityLayoutOptionsModel{
									Defaults: &observabilityLayoutDefaultsModel{Width: types.StringValue("1/2")},
								},
								Container: []observabilityGroupContainerModel{{Template: &observabilityTemplateReferenceModel{TemplateID: types.StringValue("chart-b")}}},
							},
						},
					},
				},
			},
			{
				Group: &observabilityGroupModel{
					Title:      types.StringValue("Root group"),
					Headerless: types.BoolValue(false),
					Container: []observabilityGroupContainerModel{
						{
							Layout:   &observabilityLayoutModel{Width: types.StringValue("1/2")},
							Template: &observabilityTemplateReferenceModel{TemplateID: types.StringValue("chart-c")},
						},
					},
				},
			},
		},
	}

	spec, imports, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.Equal(t, []string{"/v2/template/chart-a", "/v2/template/chart-b", "/v2/template/chart-c"}, imports)
	var document map[string]any
	require.NoError(t, json.Unmarshal(spec, &document))
	assert.Len(t, document[observabilityDashboardElement], 3)
	assert.Contains(t, document, "$import:widget0")
	assert.Contains(t, document, "$import:widget1_0_0")
	assert.Contains(t, document, "$import:widget2_0")
	layout := document["layout"].(map[string]any)
	assert.Equal(t, float64(0), layout["gap"])
	assert.Equal(t, float64(1), layout["step"])
	saved := layout["saved"].(map[string]any)
	rootItems := saved["_"].(map[string]any)["items"].([]any)
	assert.NotContains(t, rootItems[0].(map[string]any), "order")
	assert.Equal(t, true, rootItems[0].(map[string]any)["absolute"])
	assert.Equal(t, []any{"1/4", float64(8)}, rootItems[0].(map[string]any)["x"])
	sectionMetadata := saved["_.1"].(map[string]any)["section"].(map[string]any)
	assert.Equal(t, false, sectionMetadata["collapse"])
	assert.Equal(t, true, sectionMetadata["collapsible"])
	groupMetadata := saved["_.1.0"].(map[string]any)["group"].(map[string]any)
	assert.Equal(t, true, groupMetadata["headerless"])
	dashboardChildren := document[observabilityDashboardElement].([]any)
	sectionElement := dashboardChildren[1].(map[string]any)
	assert.Equal(t, float64(8), sectionElement["layout"].(map[string]any)["gap"])

	root := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{ID: "dashboard-id", Title: "Service overview", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	assert.Equal(t, model.Title, parsed.Title)
	assert.Equal(t, model.Layout, parsed.Layout)
	assert.Equal(t, model.Container, parsed.Container)
}

func TestObservabilityDashboardSpecRejectsInlineCharts(t *testing.T) {
	root := template.RootElementDashboard
	spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<Chart>":[{"<o11y:SingleValue>":[]}] }]}],"layout":{"saved":{"_":{"items":[{"id":"_.0","order":0}]}}}}`)
	_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Detail(), "typed inline charts are not supported")
}

func TestObservabilityDashboardSpecWarnsAboutUnmodeledFields(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[
			{"<Section>":[
				{"<Group>":[
					{"<Panel>":[{"<$import.widget0>":[],"futureContent":{"preserve":true}}],"futureContainer":true}
				]}
			]}
		],
		"layout":{"saved":{
			"_":{"items":[{"id":"_.0","order":0,"rootMinW":1}]},
			"_.0":{"items":[{"id":"_.0.0","order":0}],"section":{"title":"Section","futureSection":true}},
			"_.0.0":{"items":[{"id":"_.0.0.0","order":0,"minW":2,"futureConstraint":3}],"group":{"title":"Group","futureGroup":true}}
		}},
		"$import:widget0":"/v2/template/chart-id"
	}`)
	model, diags := parseDashboardTemplate(&template.Template{
		Type:       template.RecordType,
		Title:      "Dashboard",
		Spec:       spec,
		Metadata:   &template.Metadata{RootElement: &root},
		SignalView: json.RawMessage(`{"lastConvertedAt":"2026-01-01T00:00:00Z"}`),
	})
	require.False(t, diags.HasError(), diags)
	require.Len(t, model.Container, 1)
	require.NotNil(t, model.Container[0].Section)
	nestedLayout := model.Container[0].Section.Container[0].Group.Container[0].Layout
	require.NotNil(t, nestedLayout)
	assert.Equal(t, "2", nestedLayout.MinWidth.ValueString())

	warnings := diags.Warnings()
	require.Len(t, warnings, 1)
	for _, path := range []string{
		"container.0.section.container.0.group.container.0.futureContainer",
		"container.0.section.container.0.group.container.0.futureContent.preserve",
		"layout._.0.rootMinW",
		"layout._.0.0.0.futureConstraint",
		"layout.saved._.0.section.futureSection",
		"layout.saved._.0.0.group.futureGroup",
		"signalview",
	} {
		assert.Contains(t, warnings[0].Detail(), path)
	}
	assert.NotContains(t, warnings[0].Detail(), "layout._.0.0.0.minW")
}

func TestObservabilityDashboardSpecAllowsMissingLayout(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"$import:widget0":"/v2/template/chart-id"}`)
	model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	require.Len(t, model.Container, 1)
	assert.Nil(t, model.Container[0].Layout)
	assert.Equal(t, "chart-id", model.Container[0].Template.TemplateID.ValueString())
}

func TestObservabilityDashboardSpecRejectsAmbiguousLayouts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		layout string
		want   string
	}{
		"layout is not an object": {
			layout: `[]`,
			want:   "layout is []interface {} rather than an object",
		},
		"saved is not an object": {
			layout: `{"saved":[]}`,
			want:   "layout.saved is []interface {} rather than an object",
		},
		"saved list is not an object": {
			layout: `{"saved":{"_":[]}}`,
			want:   `layout.saved["_"] is []interface {} rather than an object`,
		},
		"items is not a list": {
			layout: `{"saved":{"_":{"items":{}}}}`,
			want:   `layout.saved["_"].items is map[string]interface {} rather than a list`,
		},
		"two items place one container": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0"},{"id":"_.0"}]}}}`,
			want:   "both place container 0",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"layout":` + test.layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestObservabilityDashboardSpecRejectsInvalidLayoutValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		layout string
		want   string
	}{
		"absolute is not boolean": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","absolute":"yes"}]}}}`,
			want:   "rather than a boolean",
		},
		"width is coordinate array": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","w":[1,2]}]}}}`,
			want:   "only for x and y",
		},
		"clamped width lacks value": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","w":{"min":1}}]}}}`,
			want:   "must contain value",
		},
		"coordinate contains boolean": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","x":[1,true]}]}}}`,
			want:   "coordinate part 1",
		},
		"gap is not numeric": {
			layout: `{"gap":"8","saved":{"_":{"items":[{"id":"_.0"}]}}}`,
			want:   "rather than a finite number",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"layout":` + test.layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestObservabilityDashboardSpecRejectsInvalidGroupMetadata(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[{"<Group>":[{"<Panel>":[{"<$import.widget0>":[]}]}]}],
		"layout":{"saved":{
			"_":{"items":[{"id":"_.0"}]},
			"_.0":{"items":[{"id":"_.0.0"}],"group":{"title":"Group","headerless":"yes"}}
		}},
		"$import:widget0":"/v2/template/chart-id"
	}`)
	_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.True(t, diags.HasError(), diags)
	assert.Contains(t, diags.Errors()[0].Detail(), "headerless")
	assert.Contains(t, diags.Errors()[0].Detail(), "rather than a boolean")
}

func TestObservabilityDashboardSpecAcceptsAndDiscardsUIOrder(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[
			{"<Panel>":[{"<$import.widget0>":[]}]},
			{"<Panel>":[{"<$import.widget1>":[]}]}
		],
		"layout":{"saved":{"_":{
			"at":"2026-09-11T00:00:00Z",
			"parent":"dashboard-id",
			"items":[
				{"id":"_.1","order":0.5,"w":4},
				{"id":"_.0","order":9,"w":8}
			]
		}}},
		"$import:widget0":"/v2/template/chart-a",
		"$import:widget1":"/v2/template/chart-b"
	}`)
	model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	require.Len(t, model.Container, 2)
	assert.Equal(t, "8", model.Container[0].Layout.Width.ValueString())
	assert.Equal(t, "4", model.Container[1].Layout.Width.ValueString())

	rebuilt, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.NotContains(t, string(rebuilt), `"order"`)
	assert.NotContains(t, string(rebuilt), `"parent"`)
	assert.NotContains(t, string(rebuilt), `"at"`)
}

func TestResourceObservabilityDashboardUnitTest(t *testing.T) {
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
				fwtest.WithMockResources(NewResourceObservabilityDashboard, NewResourceObservabilityTemplate),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_dashboard.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Service overview"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.width", "6/12"),
						testresource.TestCheckResourceAttrPair(
							"signalfx_observability_dashboard.test", "container.0.template.template_id",
							"signalfx_observability_template.chart", "id",
						),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_dashboard_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Updated service overview"),
						testresource.TestCheckNoResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.width"),
						testresource.TestCheckResourceAttrPair(
							"signalfx_observability_dashboard.test", "container.0.template.template_id",
							"signalfx_observability_template.chart", "id",
						),
					),
				},
			},
		},
	)
}
