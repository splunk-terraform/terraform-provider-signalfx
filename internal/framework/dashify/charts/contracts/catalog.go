// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package contracts checks curated charts against vendored product JSON
// schemas. It is used by development tools and tests, not the provider runtime.
package contracts

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

type Catalog struct {
	charts    map[string]interpreter.Chart
	contracts map[string]*jsonschema.Schema
}

// productFiles contains vendored product contracts for development checks only.
//
//go:embed O11y*.schema.json
var productFiles embed.FS

var defaultCatalog = sync.OnceValues(func() (*Catalog, error) { return Load(productFiles) })

// Default compiles the embedded product schemas once for checking tools and tests.
func Default() (*Catalog, error) { return defaultCatalog() }

// Load compiles every O11y*.schema.json in files. References resolve only against
// this catalog: loading a chart never needs a network request.
func Load(files fs.FS) (*Catalog, error) {
	paths, err := fs.Glob(files, "O11y*.schema.json")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no O11y*.schema.json chart schemas found")
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(offlineLoader{})
	catalog := &Catalog{charts: map[string]interpreter.Chart{}, contracts: map[string]*jsonschema.Schema{}}
	ids := map[string]string{}
	for _, path := range paths {
		raw, err := fs.ReadFile(files, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		document, err := interpreter.DecodeObject(string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		id, _ := document["$id"].(string)
		title, _ := document["title"].(string)
		if id == "" || !strings.HasPrefix(title, "o11y:") {
			return nil, fmt.Errorf("%s: expected $id and o11y: chart title", path)
		}
		name := chartName(strings.TrimPrefix(title, "o11y:"))
		if name == "" {
			return nil, fmt.Errorf("%s: empty chart name", path)
		}
		if _, exists := catalog.charts[name]; exists {
			return nil, fmt.Errorf("%s: duplicate chart name %q", path, name)
		}
		description, _ := document["description"].(string)
		catalog.charts[name] = interpreter.Chart{Name: name, Element: "<" + title + ">", Description: description}
		ids[name] = id
		if err := compiler.AddResource(id, document); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	for name, id := range ids {
		contract, err := compiler.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		catalog.contracts[name] = contract
	}
	return catalog, nil
}

func (c *Catalog) Charts() []interpreter.Chart {
	charts := make([]interpreter.Chart, 0, len(c.charts))
	for _, chart := range c.charts {
		charts = append(charts, chart)
	}
	sort.Slice(charts, func(i, j int) bool { return charts[i].Name < charts[j].Name })
	return charts
}

func (c *Catalog) Find(name string) (interpreter.Chart, bool) {
	chart, found := c.charts[name]
	return chart, found
}

// ValidateElement keeps curated Terraform names independent of product names.
func (c *Catalog) ValidateElement(element string, spec map[string]any) error {
	for name, chart := range c.charts {
		if chart.Element == element {
			return c.Validate(name, spec)
		}
	}
	return fmt.Errorf("no product contract for %s", element)
}

// Validate accepts an individual Dashify chart spec, including its empty element
// tag. The product schema describes the properties surrounding that tag.
func (c *Catalog) Validate(name string, spec map[string]any) error {
	chart, found := c.Find(name)
	if !found {
		return fmt.Errorf("unknown chart %q", name)
	}
	args, ok := spec[chart.Element].([]any)
	if !ok || len(args) != 0 {
		return fmt.Errorf("%s must be an empty argument array", chart.Element)
	}
	document := make(map[string]any, len(spec)-1)
	for key, value := range spec {
		if key != chart.Element {
			document[key] = value
		}
	}
	return c.contracts[name].Validate(document)
}

type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("schema reference %q is outside the embedded catalog", url)
}

func chartName(title string) string {
	if title == "TimeSeriesChart" {
		return "metrics_time_series"
	}
	title = strings.TrimSuffix(title, "Chart")
	var name strings.Builder
	runes := []rune(title)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			name.WriteByte('_')
		}
		name.WriteRune(unicode.ToLower(r))
	}
	return name.String()
}
