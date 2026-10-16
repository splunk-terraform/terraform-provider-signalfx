// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// scaffoldchart creates missing YAML drafts for vendored product JSON Schemas.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/contracts"
)

func main() {
	if len(os.Args) != 1 {
		fail(fmt.Errorf("usage: make scaffold-chart"))
	}
	base := "internal/framework/dashify/charts"
	created, skipped, err := scaffoldAll(filepath.Join(base, "contracts"), filepath.Join(base, "definitions"))
	if err != nil {
		fail(err)
	}
	for _, path := range created {
		fmt.Printf("Created %s; review its TODOs and author the supported fields.\n", path)
	}
	fmt.Printf("Created %d chart starter(s); skipped %d existing definition(s).\n", len(created), skipped)
}

func scaffoldAll(contractDir, definitionDir string) ([]string, int, error) {
	paths, err := filepath.Glob(filepath.Join(contractDir, "*.schema.json"))
	if err != nil {
		return nil, 0, err
	}
	if len(paths) == 0 {
		return nil, 0, fmt.Errorf("no *.schema.json files found in %s", contractDir)
	}
	type draft struct {
		path string
		body []byte
	}
	drafts := make([]draft, 0, len(paths))
	seen := make(map[string]string, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, 0, err
		}
		chartType, body, err := contracts.ScaffoldDefinition(raw)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", path, err)
		}
		output := filepath.Join(definitionDir, chartType+".yml")
		if previous, exists := seen[output]; exists {
			return nil, 0, fmt.Errorf("%s and %s both map to %s", previous, path, output)
		}
		seen[output] = path
		drafts = append(drafts, draft{path: output, body: body})
	}
	created, skipped := []string{}, 0
	for _, draft := range drafts {
		file, err := os.OpenFile(draft.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			skipped++
			continue
		}
		if err != nil {
			return created, skipped, err
		}
		if _, err = file.Write(draft.body); err == nil {
			err = file.Close()
		} else {
			_ = file.Close()
		}
		if err != nil {
			_ = os.Remove(draft.path)
			return created, skipped, err
		}
		created = append(created, draft.path)
	}
	return created, skipped, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
