// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"encoding/json"
	"testing"
	"testing/fstest"
)

func TestCatalogDiscoversDroppedSchemasAndValidatesOffline(t *testing.T) {
	files := fstest.MapFS{
		"O11yTimeSeriesChart.schema.json": {Data: []byte(`{"$id":"https://example.test/time","title":"o11y:TimeSeriesChart","type":"object","properties":{"program":{"type":"string"}},"required":["program"],"additionalProperties":false}`)},
		"O11ySingleValue.schema.json":     {Data: []byte(`{"$id":"https://example.test/value","title":"o11y:SingleValue","type":"object","properties":{"count":{"type":"integer"}},"required":["count"],"additionalProperties":false}`)},
	}
	catalog, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	charts := catalog.Charts()
	if len(charts) != 2 || charts[0].Name != "metrics_time_series" || charts[1].Name != "single_value" {
		t.Fatalf("chart discovery: %#v", charts)
	}
	valid := map[string]any{"<o11y:SingleValue>": []any{}, "count": json.Number("9007199254740993")}
	if err := catalog.Validate("single_value", valid); err != nil {
		t.Fatal(err)
	}
	if _, found := valid["<o11y:SingleValue>"]; !found {
		t.Fatal("validation mutated the spec")
	}
	for _, invalid := range []map[string]any{
		{"count": 1},
		{"<o11y:SingleValue>": []any{"argument"}, "count": 1},
		{"<o11y:SingleValue>": nil, "count": 1},
		{"<o11y:SingleValue>": []any{}, "count": "1"},
		{"<o11y:SingleValue>": []any{}, "count": 1.5},
		{"<o11y:SingleValue>": []any{}, "count": 1, "extra": true},
	} {
		if err := catalog.Validate("single_value", invalid); err == nil {
			t.Errorf("accepted invalid chart %#v", invalid)
		}
	}
	if err := catalog.Validate("unknown", valid); err == nil {
		t.Fatal("accepted unknown chart")
	}
}

func TestCatalogRejectsAmbiguousOrUnavailableContracts(t *testing.T) {
	for name, files := range map[string]fstest.MapFS{
		"empty catalog":    {},
		"missing identity": {"O11yFoo.schema.json": {Data: []byte(`{"type":"object"}`)}},
		"duplicate name": {
			"O11yOne.schema.json": {Data: []byte(`{"$id":"https://example.test/one","title":"o11y:FooChart"}`)},
			"O11yTwo.schema.json": {Data: []byte(`{"$id":"https://example.test/two","title":"o11y:Foo"}`)},
		},
		"external reference": {"O11yFoo.schema.json": {Data: []byte(`{"$id":"https://example.test/foo","title":"o11y:Foo","$ref":"https://offline.invalid/schema"}`)}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(files); err == nil {
				t.Fatal("accepted invalid catalog")
			}
		})
	}
}
