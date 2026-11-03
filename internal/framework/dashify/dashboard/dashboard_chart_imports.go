// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"context"
	"fmt"
	"strings"

	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
)

type dashboardChartResolver func(string) (*template.Template, error)

// Cache only within this read. Includes are optional; missing records are fetched
// individually, including when the dashboard metadata has no imports.
func newDashboardChartResolver(ctx context.Context, client *signalfx.Client, includes []template.Template) dashboardChartResolver {
	cache := map[string]*template.Template{}
	for _, record := range includes {
		id := record.ID
		if id == "" {
			id, _ = fwdashify.ObservabilityTemplateIDFromReference(record.Self)
		}
		if !fwdashify.ObservabilityTemplateIDValid(id) || record.Self != "" && record.Self != fwdashify.ObservabilityTemplateReference(id) {
			continue
		}
		record.ID = id
		cache[id] = &record
	}
	return func(id string) (*template.Template, error) {
		if !fwdashify.ObservabilityTemplateIDValid(id) {
			return nil, fmt.Errorf("invalid source chart ID %q", id)
		}
		if record := cache[id]; record != nil {
			return record, nil
		}
		result, err := client.GetTemplate(ctx, id, nil)
		if err != nil {
			return nil, fmt.Errorf("read source chart %q: %w", id, err)
		}
		record, err := fwdashify.ObservabilityTemplateFromResult(result)
		if err != nil {
			return nil, fmt.Errorf("read source chart %q: %w", id, err)
		}
		if record.ID != id || record.Self != "" && record.Self != fwdashify.ObservabilityTemplateReference(id) {
			return nil, fmt.Errorf("source chart %q returned mismatched record %q", id, record.ID)
		}
		cache[id] = record
		return record, nil
	}
}

// Only empty-argument chart imports and their Chart wrappers can be expanded
// without evaluating Dashify expressions. Other dependencies fail explicitly.
func expandDashboardChartContent(content, scope map[string]any, used map[string]bool, resolve dashboardChartResolver, visiting map[string]bool) (map[string]any, error) {
	tag, value, err := oneDashifyElement(content)
	if err != nil {
		return nil, err
	}
	if isDashifyImportElement(tag) {
		arguments, ok := value.([]any)
		if !ok || len(arguments) != 0 || len(content) != 1 {
			return nil, fmt.Errorf("import %s has arguments or sibling fields that cannot be safely embedded as inline content", tag)
		}
		alias := strings.TrimSuffix(strings.TrimPrefix(tag, dashifyImportElement), ">")
		reference, ok := scope[dashifyImportPrefix+alias].(string)
		if !ok {
			return nil, fmt.Errorf("import %q has no matching declaration", alias)
		}
		id, ok := fwdashify.ObservabilityTemplateIDFromReference(reference)
		if !ok {
			return nil, fmt.Errorf("import %q references %q rather than a local Template ID", alias, reference)
		}
		if visiting[id] {
			return nil, fmt.Errorf("cyclic chart import through %q", id)
		}
		if resolve == nil {
			return nil, fmt.Errorf("source chart %q cannot be resolved", id)
		}
		record, err := resolve(id)
		if err != nil {
			return nil, err
		}
		if record == nil || record.Metadata == nil || record.Metadata.RootElement == nil || *record.Metadata.RootElement != template.RootElementChart {
			return nil, fmt.Errorf("source template %q is not a Chart", id)
		}
		var spec map[string]any
		if decodeErr := decodeDashifyJSON(record.Spec, &spec); decodeErr != nil || spec == nil {
			return nil, fmt.Errorf("source chart %q has an invalid JSON specification", id)
		}
		if _, ok := spec[dashifyChartElement]; !ok {
			return nil, fmt.Errorf("source chart %q has no Chart element", id)
		}
		visiting[id] = true
		expanded, err := expandDashboardChartContent(spec, spec, map[string]bool{}, resolve, visiting)
		delete(visiting, id)
		if err != nil {
			return nil, fmt.Errorf("source chart %q: %w", id, err)
		}
		used[alias] = true
		return expanded, nil
	}
	if inner, ok := wrappedDashifyChartImport(content); ok {
		for key := range content {
			if key != dashifyChartElement && !strings.HasPrefix(key, dashifyImportPrefix) {
				return nil, fmt.Errorf("unsupported nested import with wrapper field %q", key)
			}
		}
		return expandDashboardChartContent(inner, scope, used, resolve, visiting)
	}
	if nestedTag, found := findDashifyImportElement(content); found {
		return nil, fmt.Errorf("unsupported nested import element %q; inline content must be self-contained", nestedTag)
	}
	if key, found := findDashifyImportDependency(content); found {
		return nil, fmt.Errorf("unsupported nested import dependency %q; inline content must be self-contained", key)
	}
	return content, nil
}

func wrappedDashifyChartImport(content map[string]any) (map[string]any, bool) {
	items, ok := content[dashifyChartElement].([]any)
	if !ok || len(items) != 1 {
		return nil, false
	}
	inner, ok := items[0].(map[string]any)
	if !ok {
		return nil, false
	}
	tag, _, err := oneDashifyElement(inner)
	if err != nil || !isDashifyImportElement(tag) {
		return nil, false
	}
	return inner, true
}

func findDashifyImportDependency(value any) (string, bool) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.HasPrefix(key, dashifyImportPrefix) || strings.HasPrefix(key, dashifyImportElement) {
				return key, true
			}
			// Dashify marks expression properties with a trailing $. Do not preserve
			// an expression while dropping the external binding that it needs.
			if expression, ok := child.(string); ok && strings.HasSuffix(key, "$") && strings.Contains(expression, "$import") {
				return key, true
			}
			if key, found := findDashifyImportDependency(child); found {
				return key, true
			}
		}
	case []any:
		for _, child := range value {
			if key, found := findDashifyImportDependency(child); found {
				return key, true
			}
		}
	}
	return "", false
}
