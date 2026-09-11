// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildObservabilityLayoutItem(id string, layout *observabilityLayoutModel) (map[string]any, error) {
	item := map[string]any{"id": id}
	if layout == nil {
		return item, nil
	}
	if !layout.Absolute.IsNull() && !layout.Absolute.IsUnknown() {
		item["absolute"] = layout.Absolute.ValueBool()
	}
	for _, field := range []struct {
		key        string
		value      types.String
		coordinate bool
	}{
		{key: "w", value: layout.Width},
		{key: "h", value: layout.Height},
		{key: "minW", value: layout.MinWidth},
		{key: "maxW", value: layout.MaxWidth},
		{key: "minH", value: layout.MinHeight},
		{key: "maxH", value: layout.MaxHeight},
		{key: "x", value: layout.X, coordinate: true},
		{key: "y", value: layout.Y, coordinate: true},
	} {
		value, ok, err := observabilityLayoutValue(field.value, field.coordinate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", field.key, err)
		}
		if ok {
			item[field.key] = value
		}
	}
	return item, nil
}

func buildObservabilityLayoutOptions(options *observabilityLayoutOptionsModel) (map[string]any, error) {
	layout := map[string]any{}
	if options == nil {
		return layout, nil
	}
	if !options.Gap.IsNull() && !options.Gap.IsUnknown() {
		layout["gap"] = options.Gap.ValueFloat64()
	}
	if !options.Step.IsNull() && !options.Step.IsUnknown() {
		layout["step"] = options.Step.ValueFloat64()
	}
	if options.Defaults == nil {
		return layout, nil
	}

	defaults := map[string]any{}
	if !options.Defaults.Absolute.IsNull() && !options.Defaults.Absolute.IsUnknown() {
		defaults["absolute"] = options.Defaults.Absolute.ValueBool()
	}
	for _, field := range []struct {
		key   string
		value types.String
	}{
		{key: "w", value: options.Defaults.Width},
		{key: "h", value: options.Defaults.Height},
		{key: "minW", value: options.Defaults.MinWidth},
		{key: "maxW", value: options.Defaults.MaxWidth},
		{key: "minH", value: options.Defaults.MinHeight},
		{key: "maxH", value: options.Defaults.MaxHeight},
	} {
		value, ok, err := observabilityLayoutValue(field.value, false)
		if err != nil {
			return nil, fmt.Errorf("defaults.%s: %w", field.key, err)
		}
		if ok {
			defaults[field.key] = value
		}
	}
	if len(defaults) > 0 {
		layout["defaults"] = defaults
	}
	return layout, nil
}

// parseObservabilityLayouts matches positional layout IDs back to one
// container list. Physical item order and the optional order field are UI
// state; IDs are the stable association with Terraform's content order.
func parseObservabilityLayouts(spec map[string]any, listKey string, count int) ([]*observabilityLayoutModel, map[string]any, []string, error) {
	layouts := make([]*observabilityLayoutModel, count)
	rawLayout, ok := spec["layout"]
	if !ok {
		return layouts, nil, nil, nil
	}
	layout, ok := rawLayout.(map[string]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("layout is %T rather than an object", rawLayout)
	}
	rawSaved, ok := layout["saved"]
	if !ok {
		return layouts, nil, nil, nil
	}
	saved, ok := rawSaved.(map[string]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("layout.saved is %T rather than an object", rawSaved)
	}
	rawContainer, ok := saved[listKey]
	if !ok {
		return layouts, nil, nil, nil
	}
	container, ok := rawContainer.(map[string]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("layout.saved[%q] is %T rather than an object", listKey, rawContainer)
	}
	takeObservabilityLayoutBookkeeping(container)
	rawItems, ok := container["items"]
	if !ok {
		return layouts, container, nil, nil
	}
	items, ok := rawItems.([]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("layout.saved[%q].items is %T rather than a list", listKey, rawItems)
	}
	delete(container, "items")

	claimed := make(map[int]string, len(items))
	var leftovers []string
	for itemNumber, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("layout item %d is %T rather than an object", itemNumber, raw)
		}
		id, ok := item["id"].(string)
		if !ok {
			return nil, nil, nil, fmt.Errorf("layout item %d has no string id", itemNumber)
		}
		delete(item, "id")
		position, err := observabilityLayoutPosition(id, listKey, count)
		if err != nil {
			return nil, nil, nil, err
		}
		if previous, exists := claimed[position]; exists {
			return nil, nil, nil, fmt.Errorf("layout items %s and %s both place container %d", previous, id, position)
		}
		claimed[position] = id
		takeObservabilityLayoutOrder(item)
		layoutModel, err := observabilityLayoutFromItem(item)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("layout %s: %w", id, err)
		}
		layouts[position] = layoutModel
		leftovers = append(leftovers, observabilityLeftovers("layout."+id, item)...)
	}
	return layouts, container, leftovers, nil
}

func takeObservabilityLayoutOrder(item map[string]any) {
	raw, ok := item["order"]
	if !ok {
		return
	}
	if number, ok := raw.(float64); ok && !math.IsNaN(number) && !math.IsInf(number, 0) {
		delete(item, "order")
	}
}

func takeObservabilityLayoutBookkeeping(entry map[string]any) {
	for _, key := range []string{"parent", "at"} {
		if _, ok := entry[key].(string); ok {
			delete(entry, key)
		}
	}
}

func observabilityLayoutPosition(id, listKey string, count int) (int, error) {
	position, ok := strings.CutPrefix(id, listKey+".")
	if !ok {
		return 0, fmt.Errorf("layout id %q is not a position under %q", id, listKey)
	}
	index, err := strconv.Atoi(position)
	if err != nil || index < 0 || index >= count {
		return 0, fmt.Errorf("layout id %q does not address a container under %q", id, listKey)
	}
	return index, nil
}

func observabilityLayoutFromItem(item map[string]any) (*observabilityLayoutModel, error) {
	absolute, err := observabilityLayoutBool(item, "absolute")
	if err != nil {
		return nil, err
	}
	width, err := observabilityLayoutString(item, "w", false)
	if err != nil {
		return nil, err
	}
	height, err := observabilityLayoutString(item, "h", false)
	if err != nil {
		return nil, err
	}
	minWidth, err := observabilityLayoutString(item, "minW", false)
	if err != nil {
		return nil, err
	}
	maxWidth, err := observabilityLayoutString(item, "maxW", false)
	if err != nil {
		return nil, err
	}
	minHeight, err := observabilityLayoutString(item, "minH", false)
	if err != nil {
		return nil, err
	}
	maxHeight, err := observabilityLayoutString(item, "maxH", false)
	if err != nil {
		return nil, err
	}
	x, err := observabilityLayoutString(item, "x", true)
	if err != nil {
		return nil, err
	}
	y, err := observabilityLayoutString(item, "y", true)
	if err != nil {
		return nil, err
	}
	if absolute.IsNull() && width.IsNull() && height.IsNull() && minWidth.IsNull() && maxWidth.IsNull() && minHeight.IsNull() && maxHeight.IsNull() && x.IsNull() && y.IsNull() {
		return nil, nil
	}
	return &observabilityLayoutModel{
		Absolute:  absolute,
		Width:     width,
		Height:    height,
		MinWidth:  minWidth,
		MaxWidth:  maxWidth,
		MinHeight: minHeight,
		MaxHeight: maxHeight,
		X:         x,
		Y:         y,
	}, nil
}

func parseObservabilityLayoutOptions(parent map[string]any) (*observabilityLayoutOptionsModel, error) {
	raw, ok := parent["layout"]
	if !ok {
		return nil, nil
	}
	layout, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("layout is %T rather than an object", raw)
	}

	gap, hasGap, err := observabilityLayoutFloat(layout, "gap")
	if err != nil {
		return nil, err
	}
	step, hasStep, err := observabilityLayoutFloat(layout, "step")
	if err != nil {
		return nil, err
	}
	defaults, err := parseObservabilityLayoutDefaults(layout)
	if err != nil {
		return nil, err
	}
	if len(layout) == 0 {
		delete(parent, "layout")
	}
	if !hasGap && !hasStep && defaults == nil {
		return nil, nil
	}
	return &observabilityLayoutOptionsModel{Gap: gap, Step: step, Defaults: defaults}, nil
}

func parseObservabilityLayoutDefaults(layout map[string]any) (*observabilityLayoutDefaultsModel, error) {
	raw, ok := layout["defaults"]
	if !ok {
		return nil, nil
	}
	defaults, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("layout.defaults is %T rather than an object", raw)
	}
	absolute, err := observabilityLayoutBool(defaults, "absolute")
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	width, err := observabilityLayoutString(defaults, "w", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	height, err := observabilityLayoutString(defaults, "h", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	minWidth, err := observabilityLayoutString(defaults, "minW", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	maxWidth, err := observabilityLayoutString(defaults, "maxW", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	minHeight, err := observabilityLayoutString(defaults, "minH", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	maxHeight, err := observabilityLayoutString(defaults, "maxH", false)
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	if len(defaults) == 0 {
		delete(layout, "defaults")
	}
	if absolute.IsNull() && width.IsNull() && height.IsNull() && minWidth.IsNull() && maxWidth.IsNull() && minHeight.IsNull() && maxHeight.IsNull() {
		return nil, nil
	}
	return &observabilityLayoutDefaultsModel{
		Absolute:  absolute,
		Width:     width,
		Height:    height,
		MinWidth:  minWidth,
		MaxWidth:  maxWidth,
		MinHeight: minHeight,
		MaxHeight: maxHeight,
	}, nil
}

func observabilityLayoutFloat(object map[string]any, key string) (types.Float64, bool, error) {
	raw, ok := object[key]
	if !ok {
		return types.Float64Null(), false, nil
	}
	number, ok := raw.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return types.Float64Null(), false, fmt.Errorf("layout.%s is %v (%T) rather than a finite number", key, raw, raw)
	}
	delete(object, key)
	return types.Float64Value(number), true, nil
}

func observabilityLayoutBool(object map[string]any, key string) (types.Bool, error) {
	raw, ok := object[key]
	if !ok {
		return types.BoolNull(), nil
	}
	value, ok := raw.(bool)
	if !ok {
		return types.BoolNull(), fmt.Errorf("has %s %v (%T) rather than a boolean", key, raw, raw)
	}
	delete(object, key)
	return types.BoolValue(value), nil
}

func buildObservabilitySectionMetadata(section *observabilitySection) map[string]any {
	metadata := map[string]any{"title": section.Title.ValueString()}
	if !section.Collapse.IsNull() && !section.Collapse.IsUnknown() {
		metadata["collapse"] = section.Collapse.ValueBool()
	}
	if !section.Collapsible.IsNull() && !section.Collapsible.IsUnknown() {
		metadata["collapsible"] = section.Collapsible.ValueBool()
	}
	return map[string]any{"section": metadata}
}

func buildObservabilityGroupMetadata(group *observabilityGroup) map[string]any {
	metadata := map[string]any{"title": group.Title.ValueString()}
	if !group.Headerless.IsNull() && !group.Headerless.IsUnknown() {
		metadata["headerless"] = group.Headerless.ValueBool()
	}
	return map[string]any{"group": metadata}
}

func parseObservabilitySectionMetadata(entry map[string]any) (types.String, types.Bool, types.Bool, error) {
	metadata, title, err := observabilityContainerMetadata(entry, "section")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), err
	}
	collapse, err := observabilityLayoutBool(metadata, "collapse")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), fmt.Errorf("layout.saved section metadata %w", err)
	}
	collapsible, err := observabilityLayoutBool(metadata, "collapsible")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), fmt.Errorf("layout.saved section metadata %w", err)
	}
	if len(metadata) == 0 {
		delete(entry, "section")
	}
	return types.StringValue(title), collapse, collapsible, nil
}

func parseObservabilityGroupMetadata(entry map[string]any) (types.String, types.Bool, error) {
	metadata, title, err := observabilityContainerMetadata(entry, "group")
	if err != nil {
		return types.StringNull(), types.BoolNull(), err
	}
	headerless, err := observabilityLayoutBool(metadata, "headerless")
	if err != nil {
		return types.StringNull(), types.BoolNull(), fmt.Errorf("layout.saved group metadata %w", err)
	}
	if len(metadata) == 0 {
		delete(entry, "group")
	}
	return types.StringValue(title), headerless, nil
}

// Section and group display state lives in each nested layout.saved entry,
// rather than beside the corresponding Dashify element.
func observabilityContainerMetadata(entry map[string]any, name string) (map[string]any, string, error) {
	raw, ok := entry[name]
	if !ok {
		return nil, "", fmt.Errorf("layout.saved entry has no %s metadata", name)
	}
	metadata, ok := raw.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("layout.saved %s metadata is %T rather than an object", name, raw)
	}
	title, ok := metadata["title"].(string)
	if !ok || title == "" {
		return nil, "", fmt.Errorf("layout.saved %s metadata has no non-empty title", name)
	}
	delete(metadata, "title")
	return metadata, title, nil
}
