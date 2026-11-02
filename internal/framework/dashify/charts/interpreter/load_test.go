// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChartDiscoveryAndExplicitOverlay(t *testing.T) {
	files := fstest.MapFS{
		"new.yml": {Data: []byte(`type: tiny
version: 1
element: 'o11y:Tiny'
extends: [common/base.yml]
properties:
  program:
    ref: common/base.yml#program
    required: false
    optional: true
    min_length: 0
  query:
    type: block
    description: Nested query.
    optional: true
    mapping: {path: [chart, query]}
    properties:
      options:
        type: block
        description: Nested options.
        optional: true
        mapping: {path: [options]}
        properties:
          style:
            type: enum
            description: Deep style.
            optional: true
            values: [line, area]
            mapping: {path: [style]}
`)},
		"common/base.yml": {Data: []byte(`version: 1
properties:
  program:
    type: string
    description: Query.
    required: true
    min_length: 1
    mapping: {path: [datasource, program]}
`)},
	}
	engine, err := load(files)
	require.NoError(t, err)
	charts := engine.Charts()
	require.Len(t, charts, 1)
	assert.Equal(t, "tiny", charts[0].Name)
	attributes, err := engine.Attributes("tiny")
	require.NoError(t, err)
	assert.False(t, attributes["program"].IsRequired(), "explicit false did not override inherited true")
	objectType := attributeTypes(attributes)
	values := map[string]attr.Value{"program": types.StringValue(""), "query": types.ObjectNull(objectType["query"].(types.ObjectType).AttrTypes)}
	value, d := types.ObjectValue(objectType, values)
	require.False(t, d.HasError(), d)
	diagnostics := engine.Validate("tiny", value)
	assert.False(t, diagnostics.HasError(), diagnostics)
}

func TestLoaderRejectsUnsafeOrIgnoredSchema(t *testing.T) {
	for name, test := range map[string]struct{ text, want string }{
		"future version":    {"type: test\nversion: 2\nelement: T\n", "schema version must be 1"},
		"unknown key":       {"type: test\nversion: 1\nelement: T\nignored: true\n", "field ignored not found"},
		"trailing document": {"type: test\nversion: 1\nelement: T\n---\nversion: 1\n", "must contain exactly one YAML document"},
		"duplicate":         {"type: test\ntype: other\n", `mapping key "type" already defined`},
		"cycle":             {"type: test\nversion: 1\nelement: T\nextends: [test.yml]\n", "cyclic extends involving test.yml"},
		"traversal":         {"type: test\nversion: 1\nelement: T\nextends: [../outside.yml]\n", `invalid schema path "../outside.yml"`},
		"ref cycle":         {"type: test\nversion: 1\nelement: T\nproperties:\n  program: {ref: 'test.yml#program'}\n", "cyclic ref test.yml#program"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := load(fstest.MapFS{"test.yml": {Data: []byte(test.text)}})
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func jsonSpec(t *testing.T, spec map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(spec)
	require.NoError(t, err)
	object, err := DecodeObject(string(data))
	require.NoError(t, err)
	return object
}

func TestTextWithoutDatasourceAndUnmodeledContent(t *testing.T) {
	engine, err := Default()
	require.NoError(t, err)
	spec := map[string]any{"<o11y:Text>": []any{}, "chart": map[string]any{"markdown": "# Notes"}, "widget": map[string]any{}}
	value, leftovers, d := engine.Decode(t.Context(), "text", jsonSpec(t, spec), types.ObjectNull(nil))
	require.False(t, d.HasError(), d)
	assert.Empty(t, leftovers, "text import without datasource left unmodeled fields")
	encoded, d := engine.Encode(t.Context(), "text", value)
	require.False(t, d.HasError(), d)
	assert.Equal(t, spec, encoded, "text round trip changed")
	for _, group := range []string{"chart", "widget"} {
		for _, malformed := range []any{nil, "invalid", []any{}} {
			invalid := jsonSpec(t, spec)
			invalid[group] = malformed
			_, _, d := engine.Decode(t.Context(), "text", invalid, value)
			assert.True(t, d.HasError(), "malformed %s accepted: %#v", group, malformed)
		}
		missing := jsonSpec(t, spec)
		delete(missing, group)
		_, _, d := engine.Decode(t.Context(), "text", missing, value)
		assert.True(t, d.HasError(), "missing required container %s accepted", group)
	}
	for _, extra := range []map[string]any{{}, {"program": "unmodeled"}} {
		withExtra := jsonSpec(t, spec)
		withExtra["datasource"] = extra
		_, leftovers, d := engine.Decode(t.Context(), "text", withExtra, value)
		want := "datasource"
		if len(extra) > 0 {
			want += ".program"
		}
		require.False(t, d.HasError(), d)
		assert.Equal(t, []string{want}, leftovers, "unmodeled datasource lost")
	}
}

func timeSeriesSpec(chart, widget map[string]any) map[string]any {
	return map[string]any{"<o11y:TimeSeriesChart>": []any{}, "chart": chart, "widget": widget, "datasource": map[string]any{"program": "data('cpu').publish()"}}
}

func TestTimeSeriesCuratedRequiredFieldsAndOmittedUIControls(t *testing.T) {
	engine, err := Default()
	require.NoError(t, err)
	attributes, err := engine.Attributes("metrics_time_series")
	require.NoError(t, err)
	assert.True(t, attributes["program"].IsRequired(), "a metric chart must require a program")
	properties := attributes["additional_properties"].(schema.ListNestedAttribute).NestedObject.Attributes
	assert.True(t, properties["property"].IsRequired(), "additional_properties.property must be required")
	assert.True(t, properties["enabled"].IsRequired(), "additional_properties.enabled must be required")
	assert.True(t, attributes["links"].(schema.ListNestedAttribute).NestedObject.Attributes["url"].IsRequired(), "links.url must be required")
	for _, name := range []string{"resolution", "backfill_slice_count"} {
		assert.NotContains(t, attributes, name, "ineffective UI control must not be exposed")
	}
}

func TestTimeSeriesLegacyFieldsRemainUnmodeled(t *testing.T) {
	engine, err := Default()
	require.NoError(t, err)
	spec := timeSeriesSpec(map[string]any{
		"chartOptions": map[string]any{"axisPrecision": int64(7), "maximumSignificantDigits": int64(5)},
		"seriesOptions": map[string]any{
			"A": map[string]any{"maximumSignificantDigits": int64(4), "displayUnit": "Byte", "precision": int64(3), "valueUnit": "Second"},
		},
	}, map[string]any{})
	value, leftovers, diagnostics := engine.Decode(t.Context(), "metrics_time_series", jsonSpec(t, spec), types.ObjectNull(nil))
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, []string{"chart.chartOptions.axisPrecision", "chart.seriesOptions.A.precision", "chart.seriesOptions.A.valueUnit"}, leftovers)
	rebuilt, diagnostics := engine.Encode(t.Context(), "metrics_time_series", value)
	want := map[string]any{
		"chartOptions": map[string]any{"maximumSignificantDigits": int64(5)},
		"seriesOptions": map[string]any{
			"A": map[string]any{"maximumSignificantDigits": int64(4), "displayUnit": "Byte"},
		},
	}
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, want, rebuilt["chart"], "modern values changed")
}

func TestTimeSeriesRejectsMalformedParentsAndUnsupportedDurations(t *testing.T) {
	engine, err := Default()
	require.NoError(t, err)
	for _, parent := range []string{"chartOptions", "legend", "noDataOptions"} {
		for _, invalid := range []any{true, "bad", []any{}, nil} {
			spec := timeSeriesSpec(map[string]any{parent: invalid}, map[string]any{})
			_, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", spec, types.ObjectNull(nil))
			assert.True(t, diagnostics.HasError(), "malformed chart.%s=%#v accepted", parent, invalid)
		}
	}
	for _, duration := range []map[string]any{
		{"type": "absolute", "start": int64(0), "end": int64(3600000)},
		{"type": "relative", "range": int64(3600000), "rangeEnd": int64(1000)},
	} {
		spec := timeSeriesSpec(map[string]any{}, map[string]any{})
		spec["datasource"].(map[string]any)["time"] = duration
		_, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", jsonSpec(t, spec), types.ObjectNull(nil))
		assert.True(t, diagnostics.HasError(), "unsupported duration accepted: %#v", duration)
	}
	value, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", timeSeriesSpec(map[string]any{}, map[string]any{}), types.ObjectNull(nil))
	require.False(t, diagnostics.HasError(), diagnostics)
	for _, invalid := range []string{"0s", "-1h", "+1h", "01h", "1.5h", "999999999999999999999999999999s"} {
		values := value.Attributes()
		values["time_range"] = types.StringValue(invalid)
		configured := types.ObjectValueMust(value.AttributeTypes(t.Context()), values)
		_, diagnostics := engine.Encode(t.Context(), "metrics_time_series", configured)
		assert.True(t, diagnostics.HasError(), "invalid duration %q accepted", invalid)
	}
}

func TestTimeSeriesReportsNestedLeftoversAtOriginalPaths(t *testing.T) {
	engine, err := Default()
	require.NoError(t, err)
	spec := timeSeriesSpec(map[string]any{
		"seriesOptions": map[string]any{"A": map[string]any{"name": "CPU", "displayUnit": map[string]any{"suffix": "%", "futureUnit": true}}},
		"yAxes":         []any{map[string]any{"min": 0.125}, map[string]any{"max": 99.5, "width": int64(50)}},
		"unknownArray":  []any{map[string]any{}, nil, []any{}, map[string]any{"nested": []any{}}},
	}, map[string]any{"links": []any{map[string]any{"url": "/service", "futureLink": nil}}})
	spec["datasource"].(map[string]any)["time"] = map[string]any{"type": "relative", "range": int64(3600000), "rangeEnd": int64(0), "futureTime": []any{}}
	value, leftovers, diagnostics := engine.Decode(t.Context(), "metrics_time_series", jsonSpec(t, spec), types.ObjectNull(nil))
	want := []string{"chart.seriesOptions.A.displayUnit.futureUnit", "chart.unknownArray.0", "chart.unknownArray.1", "chart.unknownArray.2", "chart.unknownArray.3.nested", "chart.yAxes.1.width", "datasource.time.futureTime", "widget.links.0.futureLink"}
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, want, leftovers, "nested ownership paths changed")
	rebuilt, diagnostics := engine.Encode(t.Context(), "metrics_time_series", value)
	wantSpec := timeSeriesSpec(map[string]any{
		"seriesOptions": map[string]any{"A": map[string]any{"name": "CPU", "displayUnit": map[string]any{"suffix": "%"}}},
		"yAxes":         []any{map[string]any{"min": 0.125}, map[string]any{"max": 99.5}},
	}, map[string]any{"links": []any{map[string]any{"url": "/service"}}})
	wantSpec["datasource"].(map[string]any)["time"] = map[string]any{"type": "relative", "range": int64(3600000), "rangeEnd": int64(0)}
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, wantSpec, rebuilt, "modeled nested fields did not survive the read")
}
