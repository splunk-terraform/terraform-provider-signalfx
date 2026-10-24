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
	attributes["time_range"] = types.StringValue("60m")
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
		name  string
		prior observabilityChartModel
		want  string
	}{
		{"refresh", model, "60m"},
		{"import", observabilityChartModel{}, "1h"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, diagnostics := managed.modelFromRecord(t.Context(), record, test.prior)
			require.False(t, diagnostics.HasError(), diagnostics)
			assert.Equal(t, test.want, parsed.Content.Attributes()["time_range"].(types.String).ValueString())
		})
	}
	record.Spec = json.RawMessage(`{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"data('requests').publish()","time":{"type":"relative","range":7200000,"rangeEnd":0}},"chart":{},"widget":{}}]}`)
	changed, diags := managed.modelFromRecord(t.Context(), record, model)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "2h", changed.Content.Attributes()["time_range"].(types.String).ValueString())
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
		{"null", `null`, "object"},
		{"array", `[]`, "invalid Chart specification"},
		{"trailing value", observabilityChartFixture + `{}`, "exactly one JSON object"},
		{"missing root", `{}`, "exactly one chart object"},
		{"wrong root", `{"<Dashboard>":[]}`, "exactly one chart object"},
		{"non-list root", `{"<Chart>":{}}`, "exactly one chart object"},
		{"multiple children", `{"<Chart>":[{},{}]}`, "exactly one chart object"},
		{"non-object child", `{"<Chart>":[false]}`, "not a chart object"},
		{"multiple tags", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"<o11y:Text>":[]}]}`, "multiple element keys"},
		{"unsupported tag", `{"<Chart>":[{"<o11y:Future>":[]}]}`, "unsupported element"},
		{"nonempty marker", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[true],"datasource":{"program":"A"}}]}`, "empty argument list"},
		{"missing chart group", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"widget":{}}]}`, "chart must be an object"},
		{"missing widget group", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":{}}]}`, "widget must be an object"},
		{"null group", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":null,"widget":{}}]}`, "chart must be an object"},
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

func TestObservabilityChartReadsConfigurationViolations(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	root := template.RootElementChart
	for _, test := range []struct {
		name, spec string
		want       []string
	}{
		{"missing program", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{},"chart":{},"widget":{}}]}`, []string{"1 value(s)", "program: must be set"}},
		{"invalid visualization", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":{"chartOptions":{"type":"pie"}},"widget":{}}]}`, []string{"1 value(s)", "must be one of"}},
		{"several violations", `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{},"chart":{"chartOptions":{"type":"pie"}},"widget":{}}]}`, []string{"2 value(s)", "program: must be set", "must be one of"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Title: "Chart", Spec: json.RawMessage(test.spec), Metadata: &template.Metadata{RootElement: &root}}, observabilityChartModel{})
			require.False(t, diags.HasError(), diags)
			assert.Empty(t, diags.Warnings(), "decoding alone must not apply configuration rules")
			assert.Equal(t, "chart-1", model.ID.ValueString())
			warnings := managed.ruleWarning(t.Context(), model)
			require.Len(t, warnings, 1)
			assert.Equal(t, "Stored chart violates configuration rules", warnings[0].Summary())
			for _, want := range test.want {
				assert.Contains(t, warnings[0].Detail(), want)
			}
			_, diags = managed.write(t.Context(), model)
			assert.True(t, diags.HasError(), "configuration rules must still apply to writes")
		})
	}
}

func TestObservabilityChartFromResultRejectsInvalidResponses(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	root, wrongRoot := template.RootElementChart, template.RootElementDashboard
	for _, test := range []struct {
		name   string
		result *template.Result
		want   string
	}{
		{"nil result", nil, "no template record"},
		{"nil data", &template.Result{}, "no template record"},
		{"no ID", &template.Result{Data: &template.Template{}}, "without an ID"},
		{"mismatched ID", &template.Result{Data: &template.Template{ID: "other"}}, "for Chart"},
		{"path ID", &template.Result{Data: &template.Template{ID: "../other"}}, "for Chart"},
		{"missing root", &template.Result{Data: &template.Template{ID: "chart-1", Spec: json.RawMessage(observabilityChartFixture)}}, "Chart root element"},
		{"wrong root", &template.Result{Data: &template.Template{ID: "chart-1", Metadata: &template.Metadata{RootElement: &wrongRoot}, Spec: json.RawMessage(observabilityChartFixture)}}, "Chart root element"},
		{"bad spec", &template.Result{Data: &template.Template{ID: "chart-1", Metadata: &template.Metadata{RootElement: &root}, Spec: json.RawMessage(`{}`)}}, "exactly one chart"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, diags := managed.fromResult(t.Context(), test.result, "chart-1", observabilityChartModel{})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
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
	require.True(t, managed.engine.Validate(t.Context(), model.Name, model.Content, false).HasError(), "known invalid leaf must fail despite an unrelated unknown")
	attributes["visualization"] = types.StringValue("line")
	model.Content = types.ObjectValueMust(model.Content.AttributeTypes(t.Context()), attributes)
	require.False(t, managed.engine.Validate(t.Context(), model.Name, model.Content, false).HasError())
	_, diags := managed.write(t.Context(), model)
	require.True(t, diags.HasError(), "an unresolved program must never be written")
	// A different chart's prior object must not influence duration decoding.
	prior := observabilityChartModel{Name: "text", Content: types.ObjectValueMust(map[string]attr.Type{"time_range": types.StringType}, map[string]attr.Value{"time_range": types.StringValue("60m")})}
	root := template.RootElementChart
	parsed, diags := managed.modelFromRecord(t.Context(), &template.Template{ID: "chart-1", Spec: json.RawMessage(observabilityChartFixture), Metadata: &template.Metadata{RootElement: &root}}, prior)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "1h", parsed.Content.Attributes()["time_range"].(types.String).ValueString())
}
