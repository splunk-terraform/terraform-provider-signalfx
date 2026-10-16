// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeModernFieldsAndOwnership(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{"<o11y:TimeSeriesChart>": []any{}, "chart": map[string]any{"chartOptions": map[string]any{"maximumSignificantDigits": int64(6)}, "seriesOptions": map[string]any{"cpu": map[string]any{"displayUnit": map[string]any{"prefix": "$", "suffix": " USD"}, "maximumSignificantDigits": int64(4), "unknown": map[string]any{}}}, "yAxes": []any{map[string]any{"min": 1.5}, map[string]any{"extra": nil}}}, "datasource": map[string]any{"program": "cpu"}, "widget": map[string]any{"links": []any{}}}
	original := copySpec(spec)
	value, leftovers, d := engine.Decode(context.Background(), "metrics_time_series", spec, types.ObjectNull(nil))
	if d.HasError() {
		t.Fatal(d)
	}
	if !reflect.DeepEqual(spec, original) {
		t.Fatal("Decode mutated caller input")
	}
	if !reflect.DeepEqual(leftovers, []string{"chart.seriesOptions.cpu.unknown", "chart.yAxes.1.extra"}) {
		t.Fatal(leftovers)
	}
	rebuilt, d := engine.Encode(context.Background(), "metrics_time_series", value)
	if d.HasError() {
		t.Fatal(d)
	}
	series := rebuilt["chart"].(map[string]any)["seriesOptions"].(map[string]any)["cpu"].(map[string]any)
	if !reflect.DeepEqual(series["displayUnit"], map[string]any{"prefix": "$", "suffix": " USD"}) || series["maximumSignificantDigits"] != int64(4) {
		t.Fatal(series)
	}
	if _, exists := rebuilt["widget"].(map[string]any)["links"]; !exists {
		t.Fatal("empty list changed to null")
	}
}

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
	engine, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if charts := engine.Charts(); len(charts) != 1 || charts[0].Name != "tiny" {
		t.Fatal(charts)
	}
	attributes, _ := engine.Attributes("tiny")
	if attributes["program"].IsRequired() {
		t.Fatal("explicit false did not override inherited true")
	}
	objectType := AttributeTypes(attributes)
	values := map[string]attr.Value{"program": types.StringValue(""), "query": types.ObjectNull(objectType["query"].(types.ObjectType).AttrTypes)}
	value, d := types.ObjectValue(objectType, values)
	if d.HasError() {
		t.Fatal(d)
	}
	if diagnostics := engine.Validate(context.Background(), "tiny", value, true); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
}

func TestLoaderRejectsUnsafeOrIgnoredSchema(t *testing.T) {
	for name, text := range map[string]string{
		"future version":    "type: test\nversion: 2\nelement: T\n",
		"unknown key":       "type: test\nversion: 1\nelement: T\nignored: true\n",
		"trailing document": "type: test\nversion: 1\nelement: T\n---\nversion: 1\n",
		"duplicate":         "type: test\ntype: other\n",
		"cycle":             "extends: [test.yml]\n",
		"traversal":         "extends: [../outside.yml]\n",
		"ref cycle":         "type: test\nversion: 1\nelement: T\nproperties:\n  program: {ref: 'test.yml#program'}\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(fstest.MapFS{"test.yml": {Data: []byte(text)}})
			if err == nil {
				t.Fatal("invalid schema accepted")
			}
		})
	}
}

func copySpec(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, x := range v {
			out[k] = copySpec(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = copySpec(x)
		}
		return out
	default:
		return value
	}
}

func TestTextWithoutDatasourceAndUnmodeledContent(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{"<o11y:Text>": []any{}, "chart": map[string]any{"markdown": "# Notes"}, "widget": map[string]any{}}
	value, leftovers, d := engine.Decode(t.Context(), "text", spec, types.ObjectNull(nil))
	if d.HasError() || len(leftovers) != 0 {
		t.Fatalf("text import without datasource failed: %v %v", leftovers, d)
	}
	encoded, d := engine.Encode(t.Context(), "text", value)
	if d.HasError() || !reflect.DeepEqual(encoded, spec) {
		t.Fatalf("text round trip changed: %#v %v", encoded, d)
	}
	for _, group := range []string{"chart", "widget"} {
		for _, malformed := range []any{nil, "invalid", []any{}} {
			invalid := copySpec(spec).(map[string]any)
			invalid[group] = malformed
			if _, _, d := engine.Decode(t.Context(), "text", invalid, value); !d.HasError() {
				t.Fatalf("malformed %s accepted: %#v", group, malformed)
			}
		}
		missing := copySpec(spec).(map[string]any)
		delete(missing, group)
		if _, _, d := engine.Decode(t.Context(), "text", missing, value); !d.HasError() {
			t.Fatalf("missing required container %s accepted", group)
		}
	}
	for _, extra := range []map[string]any{{}, {"program": "unmodeled"}} {
		withExtra := copySpec(spec).(map[string]any)
		withExtra["datasource"] = extra
		_, leftovers, d := engine.Decode(t.Context(), "text", withExtra, value)
		want := "datasource"
		if len(extra) > 0 {
			want += ".program"
		}
		if d.HasError() || !reflect.DeepEqual(leftovers, []string{want}) {
			t.Fatalf("unmodeled datasource lost: %v %v", leftovers, d)
		}
	}
}

func TestDefaultEngineIsSharedAndSafeForIndependentSchemaViews(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 12 {
		workers.Go(func() {
			shared, defaultError := Default()
			if defaultError != nil || shared != engine {
				t.Errorf("Default did not return the cached engine: %v", defaultError)
				return
			}
			attributes, attributesError := shared.Attributes("text")
			if attributesError != nil {
				t.Error(attributesError)
				return
			}
			delete(attributes, "markdown")
		})
	}
	workers.Wait()
	attributes, err := engine.Attributes("text")
	if err != nil || attributes["markdown"] == nil {
		t.Fatalf("schema view changed cached engine: %v", err)
	}
}

func timeSeriesSpec(chart, widget map[string]any) map[string]any {
	return map[string]any{"<o11y:TimeSeriesChart>": []any{}, "chart": chart, "widget": widget, "datasource": map[string]any{"program": "data('cpu').publish()"}}
}

func TestTimeSeriesCuratedRequiredFieldsAndOmittedUIControls(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	attributes, err := engine.Attributes("metrics_time_series")
	if err != nil {
		t.Fatal(err)
	}
	if !attributes["program"].IsRequired() {
		t.Fatal("a metric chart must require a program")
	}
	properties := attributes["additional_properties"].(schema.ListNestedAttribute).NestedObject.Attributes
	if !properties["property"].IsRequired() || !properties["enabled"].IsRequired() || !attributes["links"].(schema.ListNestedAttribute).NestedObject.Attributes["url"].IsRequired() {
		t.Fatal("configured properties and links must retain their required fields")
	}
	for _, name := range []string{"resolution", "backfill_slice_count"} {
		if _, exposed := attributes[name]; exposed {
			t.Errorf("ineffective UI control %q is exposed", name)
		}
	}
}

func TestTimeSeriesLegacyFieldsRemainUnmodeled(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	spec := timeSeriesSpec(map[string]any{
		"chartOptions": map[string]any{"axisPrecision": int64(7), "maximumSignificantDigits": int64(5)},
		"seriesOptions": map[string]any{
			"A": map[string]any{"maximumSignificantDigits": int64(4), "displayUnit": "Byte", "precision": int64(3), "valueUnit": "Second"},
		},
	}, map[string]any{})
	value, leftovers, diagnostics := engine.Decode(t.Context(), "metrics_time_series", spec, types.ObjectNull(nil))
	if diagnostics.HasError() || !reflect.DeepEqual(leftovers, []string{"chart.chartOptions.axisPrecision", "chart.seriesOptions.A.precision", "chart.seriesOptions.A.valueUnit"}) {
		t.Fatal(leftovers, diagnostics)
	}
	rebuilt, diagnostics := engine.Encode(t.Context(), "metrics_time_series", value)
	want := map[string]any{
		"chartOptions": map[string]any{"maximumSignificantDigits": int64(5)},
		"seriesOptions": map[string]any{
			"A": map[string]any{"maximumSignificantDigits": int64(4), "displayUnit": "Byte"},
		},
	}
	if diagnostics.HasError() || !reflect.DeepEqual(rebuilt["chart"], want) {
		t.Fatal("modern values changed", rebuilt, diagnostics)
	}
}

func TestTimeSeriesRejectsMalformedParentsAndUnsupportedDurations(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, parent := range []string{"chartOptions", "legend", "noDataOptions"} {
		for _, invalid := range []any{true, "bad", []any{}, nil} {
			spec := timeSeriesSpec(map[string]any{parent: invalid}, map[string]any{})
			if _, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", spec, types.ObjectNull(nil)); !diagnostics.HasError() {
				t.Errorf("malformed chart.%s=%#v accepted", parent, invalid)
			}
		}
	}
	for _, duration := range []map[string]any{
		{"type": "absolute", "start": int64(0), "end": int64(3600000)},
		{"type": "relative", "range": int64(3600000), "rangeEnd": int64(1000)},
	} {
		spec := timeSeriesSpec(map[string]any{}, map[string]any{})
		spec["datasource"].(map[string]any)["time"] = duration
		if _, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", spec, types.ObjectNull(nil)); !diagnostics.HasError() {
			t.Errorf("unsupported duration accepted: %#v", duration)
		}
	}
	value, _, diagnostics := engine.Decode(t.Context(), "metrics_time_series", timeSeriesSpec(map[string]any{}, map[string]any{}), types.ObjectNull(nil))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	for _, invalid := range []string{"0s", "-1h", "+1h", "01h", "1.5h", "999999999999999999999999999999s"} {
		values := value.Attributes()
		values["time_range"] = types.StringValue(invalid)
		configured := types.ObjectValueMust(value.AttributeTypes(t.Context()), values)
		if _, diagnostics := engine.Encode(t.Context(), "metrics_time_series", configured); !diagnostics.HasError() {
			t.Errorf("invalid duration %q accepted", invalid)
		}
	}
}

func TestTimeSeriesReportsNestedLeftoversAtOriginalPaths(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	spec := timeSeriesSpec(map[string]any{
		"seriesOptions": map[string]any{"A": map[string]any{"name": "CPU", "displayUnit": map[string]any{"suffix": "%", "futureUnit": true}}},
		"yAxes":         []any{map[string]any{"min": 0.125}, map[string]any{"max": 99.5, "width": int64(50)}},
		"unknownArray":  []any{map[string]any{}, nil, []any{}, map[string]any{"nested": []any{}}},
	}, map[string]any{"links": []any{map[string]any{"url": "/service", "futureLink": nil}}})
	spec["datasource"].(map[string]any)["time"] = map[string]any{"type": "relative", "range": int64(3600000), "rangeEnd": int64(0), "futureTime": []any{}}
	original := copySpec(spec)
	value, leftovers, diagnostics := engine.Decode(t.Context(), "metrics_time_series", spec, types.ObjectNull(nil))
	want := []string{"chart.seriesOptions.A.displayUnit.futureUnit", "chart.unknownArray.0", "chart.unknownArray.1", "chart.unknownArray.2", "chart.unknownArray.3.nested", "chart.yAxes.1.width", "datasource.time.futureTime", "widget.links.0.futureLink"}
	if diagnostics.HasError() || !reflect.DeepEqual(leftovers, want) || !reflect.DeepEqual(spec, original) {
		t.Fatal("nested ownership paths changed or input was mutated", leftovers, diagnostics)
	}
	rebuilt, diagnostics := engine.Encode(t.Context(), "metrics_time_series", value)
	wantSpec := timeSeriesSpec(map[string]any{
		"seriesOptions": map[string]any{"A": map[string]any{"name": "CPU", "displayUnit": map[string]any{"suffix": "%"}}},
		"yAxes":         []any{map[string]any{"min": 0.125}, map[string]any{"max": 99.5}},
	}, map[string]any{"links": []any{map[string]any{"url": "/service"}}})
	wantSpec["datasource"].(map[string]any)["time"] = map[string]any{"type": "relative", "range": int64(3600000), "rangeEnd": int64(0)}
	if diagnostics.HasError() || !reflect.DeepEqual(rebuilt, wantSpec) {
		t.Fatal("modeled nested fields did not survive the read", rebuilt, diagnostics)
	}
}
