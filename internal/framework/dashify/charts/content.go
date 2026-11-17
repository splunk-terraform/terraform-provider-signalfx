// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package charts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

var errUnsupportedChartType = errors.New("unsupported chart type")

// Content is chart content independent of a saved Template record.
type Content struct {
	Name  string
	Value types.Object
}

// Attributes exposes the same chart choices in standalone charts and panels.
func Attributes(engine *interpreter.Engine) (map[string]schema.Attribute, error) {
	attributes := map[string]schema.Attribute{}
	for _, chart := range engine.Charts() {
		nested, err := engine.Attributes(chart.Name)
		if err != nil {
			return nil, err
		}
		attributes[chart.Name] = schema.SingleNestedAttribute{Optional: true, MarkdownDescription: chart.Description, Attributes: nested}
	}
	return attributes, nil
}

// EncodeContent creates a complete literal Chart document, with no API calls.
func EncodeContent(ctx context.Context, engine *interpreter.Engine, content Content) (map[string]any, diag.Diagnostics) {
	value, diags := engine.Encode(ctx, content.Name, content.Value)
	if diags.HasError() {
		return nil, diags
	}
	root := "<" + engine.RootElement(content.Name) + ">"
	return map[string]any{root: []any{value}}, diags
}

// DecodeContent returns typed values for recognized charts and no chart for an
// unsupported format or type. Optional settings outside the schema's rules are omitted.
// Prior values are reused only when the chart type matches.
func DecodeContent(ctx context.Context, engine *interpreter.Engine, raw json.RawMessage, prior Content) (Content, []string, diag.Diagnostics) {
	name, chart, leftovers, err := parseContent(engine, raw)
	if errors.Is(err, errUnsupportedChartType) {
		return Content{}, nil, nil
	}
	if err != nil {
		return Content{}, nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Chart content", err.Error())}
	}
	previous := types.ObjectNull(nil)
	if prior.Name == name {
		previous = prior.Value
	}
	value, fields, diags := engine.Decode(ctx, name, chart, previous)
	return Content{Name: name, Value: value}, append(leftovers, fields...), diags
}

// parseContent identifies the chart definition and returns paths for unmodeled root fields.
func parseContent(engine *interpreter.Engine, raw json.RawMessage) (string, map[string]any, []string, error) {
	spec, err := interpreter.DecodeObject(string(raw))
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid Chart specification: %w", err)
	}
	var root string
	for key := range spec {
		if strings.HasPrefix(key, "<") {
			if root != "" {
				return "", nil, nil, fmt.Errorf("chart document has multiple root elements")
			}
			root = key
		}
	}
	if root == "" {
		return "", nil, nil, fmt.Errorf("chart document has no root element")
	}
	// Dashify permits a single child as either an object or a one-item list.
	content, ok := spec[root].(map[string]any)
	if !ok {
		children, ok := spec[root].([]any)
		if !ok || len(children) != 1 {
			return "", nil, nil, fmt.Errorf("%s must contain exactly one chart object: %w", root, errUnsupportedChartType)
		}
		content, ok = children[0].(map[string]any)
		if !ok {
			return "", nil, nil, fmt.Errorf("%s child is not a chart object: %w", root, errUnsupportedChartType)
		}
	}
	var tag string
	for key := range content {
		if strings.HasPrefix(key, "<") {
			if tag != "" {
				return "", nil, nil, fmt.Errorf("chart child has multiple element keys")
			}
			tag = key
		}
	}
	if tag == "" {
		return "", nil, nil, fmt.Errorf("chart object has no element key: %w", errUnsupportedChartType)
	}
	for _, chart := range engine.Charts() {
		if chart.Element == tag {
			delete(spec, root)
			return chart.Name, content, interpreter.Leftovers("spec", spec), nil
		}
	}
	return "", nil, nil, fmt.Errorf("chart contains unsupported element %q; use signalfx_observability_template for other chart types: %w", tag, errUnsupportedChartType)
}
