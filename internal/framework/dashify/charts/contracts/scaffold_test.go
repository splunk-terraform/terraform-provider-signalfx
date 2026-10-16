// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/definitions"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

func TestScaffoldDefinitionCreatesAnEmptyLoadableChart(t *testing.T) {
	raw, err := productFiles.ReadFile("O11yTableChart.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	chartType, draft, err := ScaffoldDefinition(raw)
	if err != nil {
		t.Fatal(err)
	}
	if chartType != "metrics_table" {
		t.Fatalf("chart type = %q, want metrics_table", chartType)
	}
	if !strings.Contains(string(draft), "element: \"o11y:TableChart\"") || !strings.Contains(string(draft), "properties: {}") || strings.Contains(string(draft), "group_by:") {
		t.Fatalf("expected an identity-only draft, got:\n%s", draft)
	}
	files := fstest.MapFS{chartType + ".yml": &fstest.MapFile{Data: draft}}
	for _, name := range []string{"common/widget.yml", "common/widget_chrome.yml", "common/widget_links.yml"} {
		body, err := definitions.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: body}
	}
	if _, err := interpreter.Load(files); err != nil {
		t.Fatalf("starter YAML does not load: %v", err)
	}
}

func TestScaffoldTextHasNoMetricsPrefix(t *testing.T) {
	raw, err := productFiles.ReadFile("O11yText.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	chartType, _, err := ScaffoldDefinition(raw)
	if err != nil {
		t.Fatal(err)
	}
	if chartType != "text" {
		t.Fatalf("chart type = %q, want text", chartType)
	}
}
