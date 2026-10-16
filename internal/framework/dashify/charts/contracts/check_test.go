// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

func TestCheckerExercisesOptionalShapesAndCuratedDomains(t *testing.T) {
	engine, err := interpreter.New()
	if err != nil {
		t.Fatal(err)
	}
	definition, found := engine.Definition("metrics_time_series")
	if !found {
		t.Fatal("missing time-series definition")
	}
	engine, err = interpreter.NewFromDefinitions([]interpreter.Definition{definition})
	if err != nil {
		t.Fatal(err)
	}
	original, err := productFiles.ReadFile("O11yTimeSeriesChart.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, definition, required, expected string
		mutate                               func(map[string]any)
	}{
		{name: "unchanged"},
		{name: "required root widget title", definition: "O11yTemplateWidget", required: "title", expected: "missing property 'title'"},
		{name: "required optional axis item minimum", definition: "O11yYAxis", required: "min", expected: "y_axes.entry.minimum"},
		{name: "required optional series map name", definition: "O11yTimeSeriesSeriesOptions", required: "name", expected: "series.entry.minimum"},
		{name: "required optional custom unit suffix", definition: "O11yDisplayUnitCustom", required: "suffix", expected: "display_unit.custom"},
		{name: "narrowed enum", definition: "O11yTimeSeriesPlotType", expected: "visualization.bar", mutate: func(d map[string]any) { d["enum"] = []any{"line", "area"} }},
		{name: "narrowed numeric upper bound", definition: "O11yTimeSeriesChartOptions", expected: "maximum_significant_digits.maximum", mutate: func(d map[string]any) {
			d["properties"].(map[string]any)["maximumSignificantDigits"].(map[string]any)["maximum"] = 19
		}},
		{name: "narrowed string length", definition: "O11ySignalFlowDatasourceConfig", expected: "minLength: got 1, want 2", mutate: func(d map[string]any) { d["properties"].(map[string]any)["program"].(map[string]any)["minLength"] = 2 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := interpreter.DecodeObject(string(original))
			if err != nil {
				t.Fatal(err)
			}
			if test.definition != "" {
				raw, exists := document["definitions"].(map[string]any)[test.definition]
				if !exists {
					t.Fatalf("missing fixture definition %s", test.definition)
				}
				definition := raw.(map[string]any)
				if test.required != "" {
					definition["required"] = []any{test.required}
				}
				if test.mutate != nil {
					test.mutate(definition)
				}
			}
			raw, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			catalog, err := Load(fstest.MapFS{"O11yTimeSeriesChart.schema.json": {Data: raw}})
			if err != nil {
				t.Fatal(err)
			}
			err = ValidateCharts(t.Context(), engine, catalog)
			if test.expected == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.expected) {
				t.Fatalf("checker result = %v, want failure identifying %q", err, test.expected)
			}
		})
	}
}

func TestConditionalSamplesSatisfyCuratedRulesOrFailClosed(t *testing.T) {
	catalog, err := Load(fstest.MapFS{"O11yConditional.schema.json": {Data: []byte(`{
		"$id":"https://example.test/conditional","title":"o11y:Conditional","type":"object",
		"properties":{"chart":{"type":"object","properties":{
			"mode":{"enum":["Range","Scale"]},"scale":{"type":"array","items":{"type":"string"}}
		},"additionalProperties":false,"dependentRequired":{"scale":["mode"]},
		"if":{"properties":{"mode":{"const":"Scale"}},"required":["mode"]},
		"then":{"required":["scale"],"properties":{"scale":{"minItems":1}}}},
		"datasource":{"type":"object"},"widget":{"type":"object"}},
		"required":["chart"],"additionalProperties":false
	}`)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, maximum := range []int{1, 0} {
		definition := interpreter.Definition{
			Type: "conditional", Version: 1, Element: "o11y:Conditional",
			Properties: map[string]interpreter.Property{
				"mode": {Type: "enum", Optional: true, Description: "Color mode.", Values: []string{"Range", "Scale"}, Mapping: interpreter.Mapping{Path: []string{"chart", "mode"}}},
				"scale": {Type: "list", Optional: true, Description: "Color buckets.", MaxItems: &maximum, Item: &interpreter.Item{Type: "string"},
					Mapping: interpreter.Mapping{Path: []string{"chart", "scale"}}, RequiresValue: &interpreter.Condition{Field: "mode", Value: "Scale"}, RequiredWhen: &interpreter.Condition{Field: "mode", Value: "Scale"}},
			},
		}
		engine, err := interpreter.NewFromDefinitions([]interpreter.Definition{definition})
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateCharts(t.Context(), engine, catalog)
		if maximum == 1 && err != nil {
			t.Fatal(err)
		}
		if maximum == 0 && (err == nil || !strings.Contains(err.Error(), "no sample can represent it")) {
			t.Fatalf("impossible conditional list did not fail closed: %v", err)
		}
	}
}

func TestEmbeddedChartsMatchProductContracts(t *testing.T) {
	engine, err := interpreter.Default()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"metrics_time_series", "metrics_single_value", "metrics_list", "metrics_table", "metrics_cluster_map", "text"} {
		if _, exists := engine.Definition(name); !exists {
			t.Fatalf("missing primary chart %s", name)
		}
	}
	if err := ValidateCharts(t.Context(), engine, catalog); err != nil {
		t.Fatal(err)
	}
}
