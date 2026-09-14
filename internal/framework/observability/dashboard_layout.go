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

// dashifyLayoutField describes one placement/size field shared by build,
// parse, validate, and empty-check logic, so the field set (schema name,
// Dashify key, coordinate handling) is declared once per model instance
// instead of separately in each of those four places. value points directly
// at the field on the caller's model, so no generics or get/set closures are
// needed to share this across dashifyLayoutModel and dashifyLayoutDefaultsModel.
type dashifyLayoutField struct {
	name       string
	key        string
	coordinate bool
	value      *types.String
}

// dashifyLayoutModelFields lists layout's placement/size fields, including
// x/y which only exist on a container's own layout, never on inherited
// defaults.
func dashifyLayoutModelFields(layout *dashifyLayoutModel) []dashifyLayoutField {
	return []dashifyLayoutField{
		{name: "width", key: "w", value: &layout.Width},
		{name: "height", key: "h", value: &layout.Height},
		{name: "min_width", key: "minW", value: &layout.MinWidth},
		{name: "max_width", key: "maxW", value: &layout.MaxWidth},
		{name: "min_height", key: "minH", value: &layout.MinHeight},
		{name: "max_height", key: "maxH", value: &layout.MaxHeight},
		{name: "x", key: "x", coordinate: true, value: &layout.X},
		{name: "y", key: "y", coordinate: true, value: &layout.Y},
	}
}

// dashifyLayoutDefaultsModelFields lists the same placement/size fields as
// dashifyLayoutModelFields, minus x/y.
func dashifyLayoutDefaultsModelFields(defaults *dashifyLayoutDefaultsModel) []dashifyLayoutField {
	return []dashifyLayoutField{
		{name: "width", key: "w", value: &defaults.Width},
		{name: "height", key: "h", value: &defaults.Height},
		{name: "min_width", key: "minW", value: &defaults.MinWidth},
		{name: "max_width", key: "maxW", value: &defaults.MaxWidth},
		{name: "min_height", key: "minH", value: &defaults.MinHeight},
		{name: "max_height", key: "maxH", value: &defaults.MaxHeight},
	}
}

func dashifyLayoutFieldsEmpty(absolute types.Bool, fields []dashifyLayoutField) bool {
	if !absolute.IsNull() {
		return false
	}
	for _, field := range fields {
		if !field.value.IsNull() {
			return false
		}
	}
	return true
}

// Writes shared layout fields into their Dashify representation.
func buildDashifyLayoutFields(target map[string]any, fields []dashifyLayoutField, errPrefix string) error {
	for _, field := range fields {
		value, ok, err := dashifyLayoutValue(*field.value, field.coordinate)
		if err != nil {
			return fmt.Errorf("%s%s: %w", errPrefix, field.key, err)
		}
		if ok {
			target[field.key] = value
		}
	}
	return nil
}

// Reads shared Dashify layout fields into a Terraform model.
func parseDashifyLayoutFields(source map[string]any, fields []dashifyLayoutField) error {
	for _, field := range fields {
		value, err := dashifyLayoutString(source, field.key, field.coordinate)
		if err != nil {
			return err
		}
		*field.value = value
	}
	return nil
}

func buildDashifyLayoutItem(id string, layout *dashifyLayoutModel) (map[string]any, error) {
	item := map[string]any{"id": id}
	if layout == nil {
		return item, nil
	}
	if !layout.Absolute.IsNull() && !layout.Absolute.IsUnknown() {
		item["absolute"] = layout.Absolute.ValueBool()
	}
	if err := buildDashifyLayoutFields(item, dashifyLayoutModelFields(layout), ""); err != nil {
		return nil, err
	}
	return item, nil
}

func buildDashifyLayoutOptions(options *dashifyLayoutOptionsModel) (map[string]any, error) {
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
	if err := buildDashifyLayoutFields(defaults, dashifyLayoutDefaultsModelFields(options.Defaults), "defaults."); err != nil {
		return nil, err
	}
	if len(defaults) > 0 {
		layout["defaults"] = defaults
	}
	return layout, nil
}

// Matches positional layout IDs to Terraform container order, discarding UI-only order state.
func parseDashifyLayouts(spec map[string]any, listKey string, count int) ([]*dashifyLayoutModel, map[string]any, []string, error) {
	layouts := make([]*dashifyLayoutModel, count)
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
	takeDashifyLayoutBookkeeping(container)
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
		position, err := dashifyLayoutPosition(id, listKey, count)
		if err != nil {
			return nil, nil, nil, err
		}
		if previous, exists := claimed[position]; exists {
			return nil, nil, nil, fmt.Errorf("layout items %s and %s both place container %d", previous, id, position)
		}
		claimed[position] = id
		takeDashifyLayoutOrder(item)
		layoutModel, err := dashifyLayoutFromItem(item)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("layout %s: %w", id, err)
		}
		layouts[position] = layoutModel
		leftovers = append(leftovers, dashifyLeftovers("layout."+id, item)...)
	}
	return layouts, container, leftovers, nil
}

func takeDashifyLayoutOrder(item map[string]any) {
	raw, ok := item["order"]
	if !ok {
		return
	}
	if number, ok := raw.(float64); ok && !math.IsNaN(number) && !math.IsInf(number, 0) {
		delete(item, "order")
	}
}

func takeDashifyLayoutBookkeeping(entry map[string]any) {
	for _, key := range []string{"parent", "at"} {
		if _, ok := entry[key].(string); ok {
			delete(entry, key)
		}
	}
}

func dashifyLayoutPosition(id, listKey string, count int) (int, error) {
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

func dashifyLayoutFromItem(item map[string]any) (*dashifyLayoutModel, error) {
	absolute, err := dashifyLayoutBool(item, "absolute")
	if err != nil {
		return nil, err
	}
	model := &dashifyLayoutModel{Absolute: absolute}
	if err := parseDashifyLayoutFields(item, dashifyLayoutModelFields(model)); err != nil {
		return nil, err
	}
	if dashifyLayoutIsEmpty(model) {
		return nil, nil
	}
	return model, nil
}

func parseDashifyLayoutOptions(parent map[string]any) (*dashifyLayoutOptionsModel, error) {
	raw, ok := parent["layout"]
	if !ok {
		return nil, nil
	}
	layout, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("layout is %T rather than an object", raw)
	}

	gap, hasGap, err := dashifyLayoutFloat(layout, "gap")
	if err != nil {
		return nil, err
	}
	step, hasStep, err := dashifyLayoutFloat(layout, "step")
	if err != nil {
		return nil, err
	}
	defaults, err := parseDashifyLayoutDefaults(layout)
	if err != nil {
		return nil, err
	}
	if len(layout) == 0 {
		delete(parent, "layout")
	}
	if !hasGap && !hasStep && defaults == nil {
		return nil, nil
	}
	return &dashifyLayoutOptionsModel{Gap: gap, Step: step, Defaults: defaults}, nil
}

func parseDashifyLayoutDefaults(layout map[string]any) (*dashifyLayoutDefaultsModel, error) {
	raw, ok := layout["defaults"]
	if !ok {
		return nil, nil
	}
	defaults, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("layout.defaults is %T rather than an object", raw)
	}
	absolute, err := dashifyLayoutBool(defaults, "absolute")
	if err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	model := &dashifyLayoutDefaultsModel{Absolute: absolute}
	if err := parseDashifyLayoutFields(defaults, dashifyLayoutDefaultsModelFields(model)); err != nil {
		return nil, fmt.Errorf("layout.defaults: %w", err)
	}
	if len(defaults) == 0 {
		delete(layout, "defaults")
	}
	if dashifyLayoutDefaultsAreEmpty(model) {
		return nil, nil
	}
	return model, nil
}

func dashifyLayoutFloat(object map[string]any, key string) (types.Float64, bool, error) {
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

func dashifyLayoutBool(object map[string]any, key string) (types.Bool, error) {
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

func buildDashifySectionMetadata(section *dashifySection) map[string]any {
	metadata := map[string]any{"title": section.Title.ValueString()}
	if !section.Collapse.IsNull() && !section.Collapse.IsUnknown() {
		metadata["collapse"] = section.Collapse.ValueBool()
	}
	if !section.Collapsible.IsNull() && !section.Collapsible.IsUnknown() {
		metadata["collapsible"] = section.Collapsible.ValueBool()
	}
	return map[string]any{"section": metadata}
}

func buildDashifyGroupMetadata(group *dashifyGroup) map[string]any {
	metadata := map[string]any{"title": group.Title.ValueString()}
	if !group.Headerless.IsNull() && !group.Headerless.IsUnknown() {
		metadata["headerless"] = group.Headerless.ValueBool()
	}
	return map[string]any{"group": metadata}
}

func parseDashifySectionMetadata(entry map[string]any) (types.String, types.Bool, types.Bool, error) {
	metadata, title, err := dashifyContainerMetadata(entry, "section")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), err
	}
	collapse, err := dashifyLayoutBool(metadata, "collapse")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), fmt.Errorf("layout.saved section metadata %w", err)
	}
	collapsible, err := dashifyLayoutBool(metadata, "collapsible")
	if err != nil {
		return types.StringNull(), types.BoolNull(), types.BoolNull(), fmt.Errorf("layout.saved section metadata %w", err)
	}
	if len(metadata) == 0 {
		delete(entry, "section")
	}
	return types.StringValue(title), collapse, collapsible, nil
}

func parseDashifyGroupMetadata(entry map[string]any) (types.String, types.Bool, error) {
	metadata, title, err := dashifyContainerMetadata(entry, "group")
	if err != nil {
		return types.StringNull(), types.BoolNull(), err
	}
	headerless, err := dashifyLayoutBool(metadata, "headerless")
	if err != nil {
		return types.StringNull(), types.BoolNull(), fmt.Errorf("layout.saved group metadata %w", err)
	}
	if len(metadata) == 0 {
		delete(entry, "group")
	}
	return types.StringValue(title), headerless, nil
}

// Section and group display state lives in layout.saved, not beside its element.
func dashifyContainerMetadata(entry map[string]any, name string) (map[string]any, string, error) {
	raw, ok := entry[name]
	if !ok {
		return nil, "", nil
	}
	if raw == nil {
		delete(entry, name)
		return nil, "", nil
	}
	metadata, ok := raw.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("layout.saved %s metadata is %T rather than an object", name, raw)
	}
	rawTitle, ok := metadata["title"]
	if !ok {
		return metadata, "", nil
	}
	if rawTitle == nil {
		delete(metadata, "title")
		return metadata, "", nil
	}
	title, ok := rawTitle.(string)
	if !ok {
		return nil, "", fmt.Errorf("layout.saved %s metadata has title %v (%T) rather than a string", name, rawTitle, rawTitle)
	}
	delete(metadata, "title")
	return metadata, title, nil
}
