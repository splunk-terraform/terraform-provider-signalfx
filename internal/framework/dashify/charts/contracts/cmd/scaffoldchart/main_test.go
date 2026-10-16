// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldAllCreatesMissingAndPreservesExisting(t *testing.T) {
	root := t.TempDir()
	contractsDir := filepath.Join(root, "contracts")
	definitionsDir := filepath.Join(root, "definitions")
	for _, dir := range []string{contractsDir, definitionsDir} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{
		"O11yNewChart.schema.json": `{"type":"object","title":"o11y:NewChart","required":["chart","datasource","widget"]}`,
		"O11yText.schema.json":     `{"type":"object","title":"o11y:Text","required":["chart","widget"]}`,
	} {
		if err := os.WriteFile(filepath.Join(contractsDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	existing := filepath.Join(definitionsDir, "text.yml")
	if err := os.WriteFile(existing, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	created, skipped, err := scaffoldAll(contractsDir, definitionsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0] != filepath.Join(definitionsDir, "metrics_new.yml") || skipped != 1 {
		t.Fatalf("created = %v, skipped = %d", created, skipped)
	}
	draft, err := os.ReadFile(created[0])
	if err != nil || !strings.Contains(string(draft), "element: \"o11y:NewChart\"") || !strings.Contains(string(draft), "properties: {}") {
		t.Fatalf("unexpected draft %q: %v", draft, err)
	}
	created, skipped, err = scaffoldAll(contractsDir, definitionsDir)
	if err != nil || len(created) != 0 || skipped != 2 {
		t.Fatalf("second run: created = %v, skipped = %d, error = %v", created, skipped, err)
	}
	unchanged, err := os.ReadFile(existing)
	if err != nil || string(unchanged) != "keep me" {
		t.Fatalf("existing definition changed: %q, %v", unchanged, err)
	}
}
