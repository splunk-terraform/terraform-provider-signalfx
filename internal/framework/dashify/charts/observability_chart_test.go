// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package charts

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
)

const observabilityChartFixture = `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"widget":{"title":"Requests"},"datasource":{"program":"data('requests').publish()","time":{"type":"relative","range":3600000,"rangeEnd":0}},"chart":{"chartOptions":{"type":"line"}}}]}`

func observabilityChartTestModel(t *testing.T) observabilityChartModel {
	t.Helper()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	root := template.RootElementChart
	model, diags := managed.modelFromRecord(t.Context(), &template.Template{
		ID: "chart-1", Title: "Request rate", Spec: json.RawMessage(observabilityChartFixture), Metadata: &template.Metadata{RootElement: &root},
	}, observabilityChartModel{})
	require.False(t, diags.HasError(), diags)
	attributes := model.Content.Attributes()
	attributes["time_range"] = types.StringValue("-60m")
	model.Content = types.ObjectValueMust(model.Content.AttributeTypes(t.Context()), attributes)
	return model
}

func TestObservabilityChartContentAndDurationRead(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	model := observabilityChartTestModel(t)
	write, diags := managed.write(t.Context(), model)
	require.False(t, diags.HasError(), diags)
	assert.JSONEq(t, observabilityChartFixture, string(write.Spec))
	assert.Equal(t, fwdashify.ObservabilityTemplateRecordType, write.Type)
	assert.Equal(t, template.RootElementChart, *write.Metadata.RootElement)
	assert.Nil(t, write.Metadata.Imports)
	assert.Nil(t, write.Metadata.Datasource)
	record := &template.Template{ID: "chart-1", Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: write.Metadata.RootElement}}
	for _, test := range []struct {
		name, configured, want string
	}{
		{"refresh", "-60m", "-60m"},
		{"compound refresh", "-30m30m", "-30m30m"},
		{"milliseconds refresh", "-3600000", "-3600000"},
		{"import", "", "-1h"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prior := observabilityChartModel{}
			if test.configured != "" {
				prior = model
				attributes := prior.Content.Attributes()
				attributes["time_range"] = types.StringValue(test.configured)
				prior.Content = types.ObjectValueMust(prior.Content.AttributeTypes(t.Context()), attributes)
				write, diagnostics := managed.write(t.Context(), prior)
				require.False(t, diagnostics.HasError(), diagnostics)
				assert.JSONEq(t, observabilityChartFixture, string(write.Spec))
			}
			parsed, diagnostics := managed.modelFromRecord(t.Context(), record, prior)
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Equal(t, test.want, parsed.Content.Attributes()["time_range"].(types.String).ValueString())
		})
	}
	record.Spec = json.RawMessage(`{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"data('requests').publish()","time":{"type":"relative","range":7200000,"rangeEnd":0}},"chart":{},"widget":{}}]}`)
	changed, diags := managed.modelFromRecord(t.Context(), record, model)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "-2h", changed.Content.Attributes()["time_range"].(types.String).ValueString())
}

func TestParseObservabilityChartSpecAcceptsUnmodeledFields(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	raw := `{"futureRoot":{"enabled":true},"chart":{},"widget":{},"datasource":{},"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"data('requests').publish()"},"chart":{"future":true},"widget":{}}]}`
	root := template.RootElementChart
	parsed, diags := managed.modelFromRecord(t.Context(), &template.Template{
		ID: "chart-1", Title: "Imported", Spec: json.RawMessage(raw), Metadata: &template.Metadata{RootElement: &root},
	}, observabilityChartModel{})
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "data('requests').publish()", parsed.Content.Attributes()["program"].(types.String).ValueString())
	require.Len(t, diags.Warnings(), 1)
	for _, name := range []string{"spec.futureRoot.enabled", "spec.chart", "spec.widget", "spec.datasource", "chart.future"} {
		assert.Contains(t, diags.Warnings()[0].Detail(), name)
	}
	assert.Contains(t, diags.Warnings()[0].Detail(), "including a title-only change")
	write, diags := managed.write(t.Context(), parsed)
	require.False(t, diags.HasError(), diags)
	assert.NotContains(t, string(write.Spec), "future")
}

func TestParseObservabilityChartSpecRejectsUnsupportedShapes(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	root := template.RootElementChart
	for _, test := range []struct{ name, spec, want string }{
		{"invalid JSON", `{"<Chart>":`, "invalid Chart specification"},
		{"missing root", `{}`, "no root element"},
		{"empty list root", `{"<Dashboard>":[]}`, "no supported chart type"},
		{"empty object root", `{"<Chart>":{}}`, "no supported chart type"},
		{"multiple children", `{"<Chart>":[{},{}]}`, "no supported chart type"},
		{"non-object child", `{"<Chart>":[false]}`, "no supported chart type"},
		{"multiple tags", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"<o11y:Text>":[]}]}`, "multiple element keys"},
		{"missing tag", `{"<Chart>":[{"chart":{}}]}`, "no supported chart type"},
		{"unsupported tag", `{"<Chart>":[{"<o11y:Future>":[]}]}`, "no supported chart type"},
		{"nonempty marker", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[true],"datasource":{"program":"A"},"chart":{},"widget":{}}]}`, "empty argument list"},
		{"missing program", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{},"chart":{},"widget":{}}]}`, "program"},
		{"empty program", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":""},"chart":{},"widget":{}}]}`, "program"},
		{"wrong program type", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":42},"chart":{},"widget":{}}]}`, "program"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Spec: json.RawMessage(test.spec), Metadata: &template.Metadata{RootElement: &root}}, observabilityChartModel{})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestObservabilityChartOmitsInvalidOptionalSettings(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	root := template.RootElementChart
	for _, test := range []struct {
		name, spec string
		want       []string
	}{
		{"invalid visualization", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":{"chartOptions":{"type":"pie"}},"widget":{}}]}`, []string{"chart.chartOptions.type"}},
		{"negative numeric controls", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A","maxDelay":-1,"minimumResolution":-1,"sampleSize":-1},"chart":{},"widget":{}}]}`, []string{"datasource.maxDelay", "datasource.minimumResolution", "datasource.sampleSize"}},
		{"missing optional groups", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"}}]}`, nil},
		{"null optional group", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":null,"widget":{}}]}`, []string{"chart"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Title: "Chart", Spec: json.RawMessage(test.spec), Metadata: &template.Metadata{RootElement: &root}}, observabilityChartModel{})
			require.False(t, diags.HasError(), diags)
			assert.Equal(t, "chart-1", model.ID.ValueString())
			if len(test.want) > 0 {
				require.Len(t, diags.Warnings(), 1)
				for _, want := range test.want {
					assert.Contains(t, diags.Warnings()[0].Detail(), want)
				}
			} else {
				assert.Empty(t, diags)
			}
			write, diags := managed.write(t.Context(), model)
			require.False(t, diags.HasError(), "decoded settings must be writable: %v", diags)
			assert.Contains(t, string(write.Spec), `"program":"A"`)
			refreshed, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Title: "Chart", Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}, model)
			require.Empty(t, diags, "written settings must remain stable on refresh")
			assert.True(t, model.Content.Equal(refreshed.Content))
		})
	}
}

func TestObservabilityChartSharedNumericControls(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	for _, chart := range managed.engine.Charts() {
		if chart.Name == "text" {
			continue
		}
		t.Run(chart.Name, func(t *testing.T) {
			base, _, diags := managed.engine.Decode(t.Context(), chart.Name, map[string]any{
				chart.Element: []any{},
				"datasource":  map[string]any{"program": "A"},
				"chart":       map[string]any{},
				"widget":      map[string]any{},
			}, types.ObjectNull(nil))
			require.False(t, diags.HasError(), diags)
			for _, field := range []string{"max_delay", "minimum_resolution", "sample_size"} {
				t.Run(field, func(t *testing.T) {
					for _, tc := range []struct {
						name    string
						value   types.Int64
						invalid bool
					}{
						{"negative", types.Int64Value(-1), true},
						{"zero", types.Int64Value(0), field == "sample_size"},
						{"positive", types.Int64Value(1), false},
						{"unset", types.Int64Null(), false},
						{"unknown", types.Int64Unknown(), false},
					} {
						t.Run(tc.name, func(t *testing.T) {
							values := base.Attributes()
							values[field] = tc.value
							value := types.ObjectValueMust(base.AttributeTypes(t.Context()), values)
							diagnostics := managed.engine.Validate(chart.Name, value)
							if tc.invalid {
								require.Len(t, diagnostics.Errors(), 1)
								assert.Contains(t, diagnostics.Errors()[0].Detail(), field+": must be at least ")
							} else {
								assert.False(t, diagnostics.HasError(), diagnostics)
							}
						})
					}
				})
			}
		})
	}
}

func TestObservabilityChartValidatesResolvedAndUnknownValues(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	model := observabilityChartTestModel(t)
	attributes := model.Content.Attributes()
	attributes["program"] = types.StringUnknown()
	attributes["visualization"] = types.StringValue("pie")
	model.Content = types.ObjectValueMust(model.Content.AttributeTypes(t.Context()), attributes)
	require.True(t, managed.engine.Validate(model.Name, model.Content).HasError(), "known invalid leaf must fail despite an unrelated unknown")
	attributes["visualization"] = types.StringValue("line")
	model.Content = types.ObjectValueMust(model.Content.AttributeTypes(t.Context()), attributes)
	require.False(t, managed.engine.Validate(model.Name, model.Content).HasError())
	// A different chart's prior object must not influence duration decoding.
	prior := observabilityChartModel{Name: "text", Content: types.ObjectValueMust(map[string]attr.Type{"time_range": types.StringType}, map[string]attr.Value{"time_range": types.StringValue("-60m")})}
	root := template.RootElementChart
	parsed, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Spec: json.RawMessage(observabilityChartFixture), Metadata: &template.Metadata{RootElement: &root}}, prior)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "-1h", parsed.Content.Attributes()["time_range"].(types.String).ValueString())
}
