// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dashifyLayoutField struct {
	name  string
	key   string
	value *types.String
}

func dashifyLayoutModelFields(layout *dashifyLayoutModel) []dashifyLayoutField {
	return []dashifyLayoutField{
		{name: "x", key: "x", value: &layout.X},
		{name: "y", key: "y", value: &layout.Y},
		{name: "width", key: "w", value: &layout.Width},
		{name: "height", key: "h", value: &layout.Height},
		{name: "min_width", key: "minW", value: &layout.MinWidth},
		{name: "min_height", key: "minH", value: &layout.MinHeight},
	}
}

func dashifyLayoutFieldsEmpty(fields []dashifyLayoutField) bool {
	for _, field := range fields {
		if !field.value.IsNull() {
			return false
		}
	}
	return true
}

// Simple sizes and positions are finite numbers or strings such as "1/2" and "100%".
func dashifyLayoutValue(value types.String) (any, bool, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, false, nil
	}
	text := value.ValueString()
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return nil, false, fmt.Errorf("JSON objects and arrays are not supported in typed layout values; use signalfx_observability_template for advanced layout")
	}
	if number, err := strconv.ParseFloat(trimmed, 64); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		return number, true, nil
	}
	return text, true, nil
}

// Leave advanced values in the source object so Read can warn about them.
func dashifyLayoutString(item map[string]any, key string) (types.String, error) {
	value, ok := item[key]
	if !ok {
		return types.StringNull(), nil
	}
	switch value := value.(type) {
	case string:
		delete(item, key)
		return types.StringValue(value), nil
	case json.Number:
		number, ok := dashifyFiniteNumber(value)
		if !ok {
			return types.StringNull(), fmt.Errorf("has invalid %s: length number %q is not finite", key, value)
		}
		delete(item, key)
		return types.StringValue(strconv.FormatFloat(number, 'f', -1, 64)), nil
	default:
		return types.StringNull(), nil
	}
}

func buildDashifyLayoutItem(id string, layout *dashifyLayoutModel) (map[string]any, error) {
	item := map[string]any{"id": id}
	if layout == nil {
		return item, nil
	}
	if !layout.Absolute.IsNull() && !layout.Absolute.IsUnknown() {
		item["absolute"] = layout.Absolute.ValueBool()
	}
	for _, field := range dashifyLayoutModelFields(layout) {
		value, set, err := dashifyLayoutValue(*field.value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", field.key, err)
		}
		if set {
			item[field.key] = value
		}
	}
	return item, nil
}

// The layout list is keyed by positional IDs rather than the order of its items.
func parseDashifyLayouts(spec map[string]any, count int) ([]*dashifyLayoutModel, []string, error) {
	layouts := make([]*dashifyLayoutModel, count)
	rawLayout, ok := spec["layout"]
	if !ok {
		return layouts, nil, nil
	}
	layout, ok := rawLayout.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("layout is %T rather than an object", rawLayout)
	}
	rawSaved, ok := layout["saved"]
	if !ok {
		pruneDashifyLayoutScaffold(spec, layout, nil, nil)
		return layouts, nil, nil
	}
	saved, ok := rawSaved.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("layout.saved is %T rather than an object", rawSaved)
	}
	rawRoot, ok := saved["_"]
	if !ok {
		pruneDashifyLayoutScaffold(spec, layout, saved, nil)
		return layouts, nil, nil
	}
	root, ok := rawRoot.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("layout.saved[%q] is %T rather than an object", "_", rawRoot)
	}
	for _, key := range []string{"parent", "at"} {
		if _, isString := root[key].(string); isString {
			delete(root, key)
		}
	}
	rawItems, ok := root["items"]
	if !ok {
		pruneDashifyLayoutScaffold(spec, layout, saved, root)
		return layouts, nil, nil
	}
	items, ok := rawItems.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("layout.saved[%q].items is %T rather than a list", "_", rawItems)
	}
	delete(root, "items")
	claimed := make(map[int]string, len(items))
	var leftovers []string
	for itemNumber, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("layout item %d is %T rather than an object", itemNumber, raw)
		}
		id, ok := item["id"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("layout item %d has no string id", itemNumber)
		}
		delete(item, "id")
		positionText, ok := strings.CutPrefix(id, "_.")
		if !ok {
			return nil, nil, fmt.Errorf("layout id %q is not a position under %q", id, "_")
		}
		position, err := strconv.Atoi(positionText)
		if err != nil || position < 0 || position >= count {
			return nil, nil, fmt.Errorf("layout id %q does not address a container under %q", id, "_")
		}
		if previous, exists := claimed[position]; exists {
			return nil, nil, fmt.Errorf("layout items %s and %s both place container %d", previous, id, position)
		}
		claimed[position] = id
		model := &dashifyLayoutModel{}
		if absolute, ok := item["absolute"].(bool); ok {
			model.Absolute = types.BoolValue(absolute)
			delete(item, "absolute")
		}
		for _, field := range dashifyLayoutModelFields(model) {
			value, err := dashifyLayoutString(item, field.key)
			if err != nil {
				return nil, nil, fmt.Errorf("layout %s: %w", id, err)
			}
			*field.value = value
		}
		for key, value := range item {
			if object, ok := value.(map[string]any); ok && len(object) == 0 {
				return nil, nil, fmt.Errorf("layout %s has unsupported empty %s layout object", id, key)
			}
		}
		if !model.Absolute.IsNull() || !dashifyLayoutFieldsEmpty(dashifyLayoutModelFields(model)) {
			layouts[position] = model
		}
		leftovers = append(leftovers, dashifyLeftovers("layout."+id, item)...)
	}
	pruneDashifyLayoutScaffold(spec, layout, saved, root)
	return layouts, leftovers, nil
}

func pruneDashifyLayoutScaffold(spec, layout, saved, root map[string]any) {
	if root != nil && len(root) == 0 {
		delete(saved, "_")
	}
	if saved != nil && len(saved) == 0 {
		delete(layout, "saved")
	}
	if len(layout) == 0 {
		delete(spec, "layout")
	}
}
