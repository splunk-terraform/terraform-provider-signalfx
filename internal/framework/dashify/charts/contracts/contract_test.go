// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/definitions"
)

func TestContractIdentity(t *testing.T) {
	catalog := discoverContracts(t)
	for _, definition := range catalog {
		document := readContract(t, definition)
		if document["$schema"] != "https://json-schema.org/draft/2020-12/schema" || document["$id"] != definition.SchemaID || "<"+document["title"].(string)+">" != definition.Element {
			t.Errorf("%s catalog identity does not match product schema", definition.File)
		}
	}
}

func TestCuratedDefinitionsMatchDiscoveredProductContracts(t *testing.T) {
	catalog := discoverContracts(t)
	paths, err := fs.Glob(definitions.Files, "*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no curated chart definitions")
	}
	names, used := map[string]bool{}, map[string]bool{}
	for _, path := range paths {
		raw, err := definitions.Files.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var definition struct {
			Type    string `yaml:"type"`
			Version int    `yaml:"version"`
			Element string `yaml:"element"`
		}
		if err := yaml.Unmarshal(raw, &definition); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if definition.Version != 1 || definition.Type == "" || definition.Type+".yml" != path || names[definition.Type] {
			t.Fatalf("invalid or duplicate chart identity in %s", path)
		}
		names[definition.Type] = true
		element := "<" + definition.Element + ">"
		if !hasElement(catalog, element) || used[element] {
			t.Fatalf("%s has missing or duplicate product contract %s", path, element)
		}
		used[element] = true
	}
	for _, definition := range catalog {
		if !used[definition.Element] {
			t.Errorf("%s has no curated chart definition", definition.File)
		}
	}
}

func TestContractsCompileOfflineAndRequireTheirRootGroups(t *testing.T) {
	for _, definition := range discoverContracts(t) {
		t.Run(definition.File, func(t *testing.T) {
			document := readContract(t, definition)
			contract := compileContract(t, definition, document)
			required, ok := document["required"].([]any)
			if !ok || len(required) == 0 {
				t.Fatal("contract has no required root groups")
			}
			minimal := map[string]any{}
			for _, group := range required {
				minimal[group.(string)] = map[string]any{}
			}
			if err := contract.Validate(minimal); err != nil {
				t.Fatalf("minimal product shape rejected: %v", err)
			}
			for group := range minimal {
				without := clone(minimal)
				delete(without, group)
				if err := contract.Validate(without); err == nil {
					t.Errorf("missing %s accepted", group)
				}
			}
			for name, invalid := range map[string]map[string]any{
				"chart type":   with(minimal, "chart", []any{}),
				"widget title": with(minimal, "widget", map[string]any{"title": true}),
				"unknown root": with(minimal, "unknown", true),
			} {
				if err := contract.Validate(invalid); err == nil {
					t.Errorf("invalid %s accepted", name)
				}
			}
		})
	}
}

type contractDefinition struct {
	File, Element, SchemaID string
}

func discoverContracts(t *testing.T) []contractDefinition {
	t.Helper()
	paths, err := fs.Glob(productFiles, "O11y*.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no embedded product contracts")
	}
	catalog := make([]contractDefinition, 0, len(paths))
	for _, path := range paths {
		definition := contractDefinition{File: path}
		document := readContract(t, definition)
		id, validID := document["$id"].(string)
		title, validTitle := document["title"].(string)
		if !validID || id == "" || !validTitle || !strings.HasPrefix(title, "o11y:") {
			t.Fatalf("%s needs a schema ID and o11y: title", path)
		}
		definition.SchemaID, definition.Element = id, "<"+title+">"
		catalog = append(catalog, definition)
	}
	return catalog
}

func hasElement(catalog []contractDefinition, element string) bool {
	for _, definition := range catalog {
		if definition.Element == element {
			return true
		}
	}
	return false
}

func readContract(t *testing.T, definition contractDefinition) map[string]any {
	t.Helper()
	raw, err := productFiles.ReadFile(definition.File)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func compileContract(t *testing.T, definition contractDefinition, document map[string]any) *jsonschema.Schema {
	t.Helper()
	var inspect func(any)
	inspect = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if ref, exists := value["$ref"]; exists {
				pointer, ok := ref.(string)
				if !ok || !strings.HasPrefix(pointer, "#/definitions/") {
					t.Fatalf("unexpected non-local schema ref: %v", ref)
				}
			}
			for _, child := range value {
				inspect(child)
			}
		case []any:
			for _, child := range value {
				inspect(child)
			}
		}
	}
	inspect(document)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(definition.SchemaID, document); err != nil {
		t.Fatal(err)
	}
	contract, err := compiler.Compile(definition.SchemaID)
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func clone(original map[string]any) map[string]any {
	result := make(map[string]any, len(original))
	for key, value := range original {
		result[key] = value
	}
	return result
}

func with(original map[string]any, key string, value any) map[string]any {
	result := clone(original)
	result[key] = value
	return result
}
