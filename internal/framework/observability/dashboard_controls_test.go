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
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestObservabilityControlBarSchemaAndModel(t *testing.T) {
	t.Parallel()

	resourceSchema := observabilityControlBarBlock()
	assert.NotEmpty(t, resourceSchema.Description)
	for _, name := range []string{"time_range", "density", "pinned_filter", "filter_set"} {
		assert.Contains(t, resourceSchema.Blocks, name)
	}
	pinned, ok := resourceSchema.Blocks["pinned_filter"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Contains(t, pinned.NestedObject.Attributes, "variable_name")
	filterSet, ok := resourceSchema.Blocks["filter_set"].(schema.SingleNestedBlock)
	require.True(t, ok)
	filter, ok := filterSet.Blocks["filter"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Contains(t, filter.NestedObject.Attributes, "values")
}

func TestBuildAndParseObservabilityControlBar(t *testing.T) {
	t.Parallel()

	model := &observabilityControlBarModel{
		TimeRange: &observabilityTimeRangeControlModel{
			Label:                types.StringValue("Time Range"),
			Description:          types.StringValue("Dashboard window"),
			Hidden:               types.BoolValue(false),
			DefaultVariableValue: types.StringValue("-PT15M"),
		},
		Density: &observabilityDensityControlModel{
			Label:                types.StringValue("Density"),
			Description:          types.StringValue("Resolution"),
			Hidden:               types.BoolValue(false),
			DefaultVariableValue: types.Int64Value(60),
		},
		PinnedFilter: []observabilityPinnedFilterControlModel{
			{
				VariableName:               types.StringValue("service"),
				Label:                      types.StringValue("Service"),
				Description:                types.StringValue("Service to inspect"),
				Hidden:                     types.BoolValue(false),
				Key:                        types.StringValue("service.name"),
				DefaultVariableValue:       []types.String{types.StringValue("checkout")},
				SuggestedValues:            []types.String{types.StringValue("checkout"), types.StringValue("payments")},
				OnlySuggestPreferredValues: types.BoolValue(true),
				MatchMissing:               types.BoolValue(false),
				Required:                   types.BoolValue(true),
				ApplicationMode:            types.StringValue("add"),
			},
			{
				VariableName:         types.StringValue("environment"),
				DefaultVariableValue: []types.String{},
			},
		},
		FilterSet: &observabilityFilterSetControlModel{
			Label:       types.StringValue("Filters"),
			Description: types.StringValue("Ad-hoc filters"),
			Hidden:      types.BoolValue(false),
			Filter: []observabilityFilterSetEntryModel{
				{
					Key:      types.StringValue("deployment.environment"),
					Values:   []types.String{types.StringValue("prod")},
					Negated:  types.BoolValue(false),
					Disabled: types.BoolValue(true),
				},
				{Key: types.StringValue("region"), Values: []types.String{}},
			},
		},
	}

	built := buildObservabilityControlBar(model)
	controls, ok := built["controls"].([]any)
	require.True(t, ok)
	require.Len(t, controls, 5)
	assert.Equal(t, []any{"TimeRange", "Density", "PinnedFilter", "PinnedFilter", "FilterSet"}, controlTypes(controls))
	assert.Equal(t, []any{"TIME", "DENSITY", "service", "environment", "FILTERS"}, controlVariableNames(controls))

	raw, err := json.Marshal(built)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	parsed, leftovers, err := parseObservabilityControlBar(map[string]any{"controlBar": decoded})
	require.NoError(t, err)
	assert.Empty(t, leftovers)
	require.NotNil(t, parsed)
	assert.Equal(t, "-PT15M", parsed.TimeRange.DefaultVariableValue.ValueString())
	assert.Equal(t, int64(60), parsed.Density.DefaultVariableValue.ValueInt64())
	assert.Equal(t, false, parsed.TimeRange.Hidden.ValueBool())
	assert.Equal(t, false, parsed.PinnedFilter[0].MatchMissing.ValueBool())
	assert.NotNil(t, parsed.PinnedFilter[1].DefaultVariableValue)
	assert.Empty(t, parsed.PinnedFilter[1].DefaultVariableValue)
	assert.NotNil(t, parsed.FilterSet.Filter)
	assert.Empty(t, parsed.FilterSet.Filter[1].Values)
}

func TestParseObservabilityControlBarIndependentJSON(t *testing.T) {
	t.Parallel()

	spec := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(`{
		"controlBar": {"controls": [
			{"type":"PinnedFilter","variableName":"host","key":"host.name","defaultVariableValue":["web-1"],"preferablySuggestedValues":["web-1","web-2"],"onlySuggestPreferredValues":true,"matchMissing":false,"required":true,"applicationMode":"override"},
			{"type":"TimeRange","variableName":"TIME","defaultVariableValue":"2026-01-01T00:00:00Z--2026-01-01T01:00:00Z"},
			{"type":"FilterSet","variableName":"FILTERS","defaultVariableValue":[{"key":"env","values":[],"negated":true,"disabled":false}]}
		]}
	}`), &spec))

	model, leftovers, err := parseObservabilityControlBar(spec)
	require.NoError(t, err)
	assert.Empty(t, leftovers)
	require.NotNil(t, model)
	assert.Equal(t, "host", model.PinnedFilter[0].VariableName.ValueString())
	assert.Equal(t, "2026-01-01T00:00:00Z--2026-01-01T01:00:00Z", model.TimeRange.DefaultVariableValue.ValueString())
	assert.Equal(t, "override", model.PinnedFilter[0].ApplicationMode.ValueString())
	assert.True(t, model.FilterSet.Filter[0].Negated.ValueBool())
	assert.Empty(t, spec)
}

func TestObservabilityControlBarCodecOmissionAndWarnings(t *testing.T) {
	t.Parallel()

	model, leftovers, err := parseObservabilityControlBar(map[string]any{})
	require.NoError(t, err)
	assert.Nil(t, model)
	assert.Empty(t, leftovers)

	spec := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(`{"controlBar":{"controls":[
		{"type":"Future","variableName":"future","future":true},
		{"type":"Density","variableName":"wrong","future":true},
		{"type":"PinnedFilter","variableName":"FILTERS","future":true},
		{"type":"TimeRange","variableName":"TIME","future":true,"label":"Time","nested":{"x":1}}
	],"futureBar":true}}`), &spec))
	parsed, leftovers, err := parseObservabilityControlBar(spec)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Contains(t, leftovers, "controlBar.controls.0")
	assert.Contains(t, leftovers, "controlBar.controls.1")
	assert.Contains(t, leftovers, "controlBar.controls.2")
	assert.Contains(t, leftovers, "controlBar.controls.3.nested.x")
	assert.Contains(t, leftovers, "controlBar.futureBar")

	filterSpec := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(`{"controlBar":{"controls":[
		{"type":"FilterSet","variableName":"FILTERS","defaultVariableValue":[{"key":"env","values":[],"futureFilter":true}]}
	]}}`), &filterSpec))
	_, leftovers, err = parseObservabilityControlBar(filterSpec)
	require.NoError(t, err)
	assert.Contains(t, leftovers, "controlBar.controls.0.defaultVariableValue.0.futureFilter")
}

func TestObservabilityControlBarCodecErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec string
		want string
	}{
		"control bar is not object": {spec: `{"controlBar":[]}`, want: "rather than an object"},
		"controls is not list":      {spec: `{"controlBar":{"controls":{}}}`, want: "rather than a list"},
		"control is not object":     {spec: `{"controlBar":{"controls":[true]}}`, want: "rather than an object"},
		"known field wrong type":    {spec: `{"controlBar":{"controls":[{"type":"Density","variableName":"DENSITY","defaultVariableValue":"60"}]}}`, want: "rather than an integer"},
		"filter key missing":        {spec: `{"controlBar":{"controls":[{"type":"FilterSet","variableName":"FILTERS","defaultVariableValue":[{"values":[]}] }]}}`, want: "must be a non-empty string"},
		"filter values missing":     {spec: `{"controlBar":{"controls":[{"type":"FilterSet","variableName":"FILTERS","defaultVariableValue":[{"key":"env"}] }]}}`, want: "values must be a list"},
		"duplicate pinned":          {spec: `{"controlBar":{"controls":[{"type":"PinnedFilter","variableName":"env"},{"type":"PinnedFilter","variableName":"env"}]}}`, want: "duplicates pinned filter"},
		"duplicate singleton":       {spec: `{"controlBar":{"controls":[{"type":"Density","variableName":"DENSITY"},{"type":"Density","variableName":"DENSITY"}]}}`, want: "duplicates the canonical Density"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var spec map[string]any
			require.NoError(t, json.Unmarshal([]byte(test.spec), &spec))
			_, _, err := parseObservabilityControlBar(spec)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestValidateObservabilityControlBar(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model *observabilityControlBarModel
		want  string
	}{
		"empty":     {model: &observabilityControlBarModel{}, want: "at least one"},
		"reserved":  {model: &observabilityControlBarModel{PinnedFilter: []observabilityPinnedFilterControlModel{{VariableName: types.StringValue("TIME")}}}, want: "reserved"},
		"duplicate": {model: &observabilityControlBarModel{PinnedFilter: []observabilityPinnedFilterControlModel{{VariableName: types.StringValue("env")}, {VariableName: types.StringValue("env")}}}, want: "already uses"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var response resource.ValidateConfigResponse
			validateObservabilityControlBar(&response, path.Root("control_bar"), test.model)
			require.True(t, response.Diagnostics.HasError())
			assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.want)
		})
	}
}

func TestObservabilityControlBarAttributeValidators(t *testing.T) {
	t.Parallel()

	bar := observabilityControlBarBlock()
	density := bar.Blocks["density"].(schema.SingleNestedBlock).Attributes["default_variable_value"].(schema.Int64Attribute)
	assertValidatorInt64(t, density.Validators, 60, false)
	assertValidatorInt64(t, density.Validators, 45, true)

	pinned := bar.Blocks["pinned_filter"].(schema.ListNestedBlock).NestedObject.Attributes
	applicationMode := pinned["application_mode"].(schema.StringAttribute)
	assertValidatorString(t, applicationMode.Validators, "override", false)
	assertValidatorString(t, applicationMode.Validators, "replace_only", true)
}

func TestResourceObservabilityDashboardAllControlsConfig(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
			t,
			nil,
			fwtest.WithMockResources(NewResourceObservabilityDashboard, NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{{
			ConfigFile:         config.StaticFile("testdata/observability_dashboard_controls.tf"),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestBuildDashboardSpecOmitsAbsentControlBar(t *testing.T) {
	t.Parallel()

	raw, _, err := buildDashboardSpec(observabilityDashboardModel{Title: types.StringValue("No controls")})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "controlBar")
}

func TestDashboardSpecPlacesControlBarAtTopLevel(t *testing.T) {
	t.Parallel()

	controlBar := &observabilityControlBarModel{
		TimeRange: &observabilityTimeRangeControlModel{DefaultVariableValue: types.StringValue("-15m")},
	}
	raw, _, err := buildDashboardSpec(observabilityDashboardModel{
		Title:      types.StringValue("Controls"),
		ControlBar: controlBar,
	})
	require.NoError(t, err)
	var spec map[string]any
	require.NoError(t, json.Unmarshal(raw, &spec))
	assert.Contains(t, spec, "controlBar")
	assert.Contains(t, spec, "<Dashboard>")
	assert.Contains(t, spec, "layout")

	root := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{Title: "Controls", Spec: raw, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	require.NotNil(t, parsed.ControlBar)
	require.NotNil(t, parsed.ControlBar.TimeRange)
	assert.Equal(t, "-15m", parsed.ControlBar.TimeRange.DefaultVariableValue.ValueString())
}

func assertValidatorInt64(t *testing.T, validators []validator.Int64, value int64, wantError bool) {
	t.Helper()
	response := validator.Int64Response{}
	validators[0].ValidateInt64(context.Background(), validator.Int64Request{ConfigValue: types.Int64Value(value)}, &response)
	assert.Equal(t, wantError, response.Diagnostics.HasError())
}

func assertValidatorString(t *testing.T, validators []validator.String, value string, wantError bool) {
	t.Helper()
	response := validator.StringResponse{}
	validators[0].ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, &response)
	assert.Equal(t, wantError, response.Diagnostics.HasError())
}

func controlTypes(controls []any) []any {
	result := make([]any, len(controls))
	for i, control := range controls {
		result[i] = control.(map[string]any)["type"]
	}
	return result
}

func controlVariableNames(controls []any) []any {
	result := make([]any, len(controls))
	for i, control := range controls {
		result[i] = control.(map[string]any)["variableName"]
	}
	return result
}
