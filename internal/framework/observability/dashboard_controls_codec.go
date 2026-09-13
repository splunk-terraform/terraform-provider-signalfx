// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	observabilityTimeRangeVariableName = "TIME"
	observabilityDensityVariableName   = "DENSITY"
	observabilityFilterSetVariableName = "FILTERS"
)

func buildObservabilityControlBar(model *observabilityControlBarModel) map[string]any {
	controls := make([]any, 0)
	if model == nil {
		return map[string]any{"controls": controls}
	}
	if model.TimeRange != nil {
		control := map[string]any{
			"type":         "TimeRange",
			"variableName": observabilityTimeRangeVariableName,
		}
		setObservabilityControlString(control, "label", model.TimeRange.Label)
		setObservabilityControlString(control, "description", model.TimeRange.Description)
		setObservabilityControlBool(control, "hidden", model.TimeRange.Hidden)
		setObservabilityControlString(control, "defaultVariableValue", model.TimeRange.DefaultVariableValue)
		controls = append(controls, control)
	}
	if model.Density != nil {
		control := map[string]any{
			"type":         "Density",
			"variableName": observabilityDensityVariableName,
		}
		setObservabilityControlString(control, "label", model.Density.Label)
		setObservabilityControlString(control, "description", model.Density.Description)
		setObservabilityControlBool(control, "hidden", model.Density.Hidden)
		if !model.Density.DefaultVariableValue.IsNull() && !model.Density.DefaultVariableValue.IsUnknown() {
			control["defaultVariableValue"] = model.Density.DefaultVariableValue.ValueInt64()
		}
		controls = append(controls, control)
	}
	for _, filter := range model.PinnedFilter {
		control := map[string]any{
			"type":         "PinnedFilter",
			"variableName": filter.VariableName.ValueString(),
		}
		setObservabilityControlString(control, "label", filter.Label)
		setObservabilityControlString(control, "description", filter.Description)
		setObservabilityControlBool(control, "hidden", filter.Hidden)
		setObservabilityControlString(control, "key", filter.Key)
		setObservabilityControlStringList(control, "defaultVariableValue", filter.DefaultVariableValue)
		setObservabilityControlStringList(control, "preferablySuggestedValues", filter.SuggestedValues)
		setObservabilityControlBool(control, "onlySuggestPreferredValues", filter.OnlySuggestPreferredValues)
		setObservabilityControlBool(control, "matchMissing", filter.MatchMissing)
		setObservabilityControlBool(control, "required", filter.Required)
		setObservabilityControlString(control, "applicationMode", filter.ApplicationMode)
		controls = append(controls, control)
	}
	if model.FilterSet != nil {
		control := map[string]any{
			"type":         "FilterSet",
			"variableName": observabilityFilterSetVariableName,
		}
		setObservabilityControlString(control, "label", model.FilterSet.Label)
		setObservabilityControlString(control, "description", model.FilterSet.Description)
		setObservabilityControlBool(control, "hidden", model.FilterSet.Hidden)
		entries := make([]any, len(model.FilterSet.Filter))
		for i, filter := range model.FilterSet.Filter {
			entry := map[string]any{
				"key":    filter.Key.ValueString(),
				"values": observabilityControlStringsToAny(filter.Values),
			}
			setObservabilityControlBool(entry, "negated", filter.Negated)
			setObservabilityControlBool(entry, "disabled", filter.Disabled)
			entries[i] = entry
		}
		control["defaultVariableValue"] = entries
		controls = append(controls, control)
	}
	return map[string]any{"controls": controls}
}

func setObservabilityControlString(destination map[string]any, key string, value types.String) {
	if !value.IsNull() && !value.IsUnknown() {
		destination[key] = value.ValueString()
	}
}

func setObservabilityControlBool(destination map[string]any, key string, value types.Bool) {
	if !value.IsNull() && !value.IsUnknown() {
		destination[key] = value.ValueBool()
	}
}

func setObservabilityControlStringList(destination map[string]any, key string, values []types.String) {
	if values == nil {
		return
	}
	destination[key] = observabilityControlStringsToAny(values)
}

func observabilityControlStringsToAny(values []types.String) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
}

func parseObservabilityControlBar(spec map[string]any) (*observabilityControlBarModel, []string, error) {
	raw, ok := spec["controlBar"]
	if !ok {
		return nil, nil, nil
	}
	delete(spec, "controlBar")
	bar, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("controlBar is %T rather than an object", raw)
	}
	rawControls, ok := bar["controls"]
	if !ok {
		return nil, nil, fmt.Errorf("controlBar has no controls list")
	}
	delete(bar, "controls")
	controls, ok := rawControls.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("controlBar.controls is %T rather than a list", rawControls)
	}

	model := &observabilityControlBarModel{}
	seenPinned := map[string]struct{}{}
	seenSingleton := map[string]bool{}
	var leftovers []string
	for i, rawControl := range controls {
		controlPath := fmt.Sprintf("controlBar.controls.%d", i)
		control, ok := rawControl.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%s is %T rather than an object", controlPath, rawControl)
		}
		controlType, err := takeObservabilityControlString(control, "type", controlPath)
		if err != nil {
			return nil, nil, err
		}
		variableName, err := takeObservabilityControlString(control, "variableName", controlPath)
		if err != nil {
			return nil, nil, err
		}
		if variableName.IsNull() || variableName.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.variableName must be a non-empty string", controlPath)
		}
		if controlType.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.type must be a non-empty string", controlPath)
		}

		switch controlType.ValueString() {
		case "TimeRange":
			if variableName.ValueString() != observabilityTimeRangeVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			if seenSingleton[observabilityTimeRangeVariableName] {
				return nil, nil, fmt.Errorf("%s duplicates the canonical TimeRange control", controlPath)
			}
			seenSingleton[observabilityTimeRangeVariableName] = true
			value := &observabilityTimeRangeControlModel{}
			if value.Label, err = takeObservabilityControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeObservabilityControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeObservabilityControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeObservabilityControlString(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.TimeRange = value
		case "Density":
			if variableName.ValueString() != observabilityDensityVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			if seenSingleton[observabilityDensityVariableName] {
				return nil, nil, fmt.Errorf("%s duplicates the canonical Density control", controlPath)
			}
			seenSingleton[observabilityDensityVariableName] = true
			value := &observabilityDensityControlModel{}
			if value.Label, err = takeObservabilityControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeObservabilityControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeObservabilityControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeObservabilityControlInt64(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.Density = value
		case "PinnedFilter":
			if variableName.ValueString() == observabilityTimeRangeVariableName || variableName.ValueString() == observabilityDensityVariableName || variableName.ValueString() == observabilityFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			if _, exists := seenPinned[variableName.ValueString()]; exists {
				return nil, nil, fmt.Errorf("%s duplicates pinned filter variableName %q", controlPath, variableName.ValueString())
			}
			seenPinned[variableName.ValueString()] = struct{}{}
			value := &observabilityPinnedFilterControlModel{VariableName: variableName}
			if value.Label, err = takeObservabilityControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeObservabilityControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeObservabilityControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Key, err = takeObservabilityControlString(control, "key", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeObservabilityControlStringList(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			if value.SuggestedValues, err = takeObservabilityControlStringList(control, "preferablySuggestedValues", controlPath); err != nil {
				return nil, nil, err
			}
			if value.OnlySuggestPreferredValues, err = takeObservabilityControlBool(control, "onlySuggestPreferredValues", controlPath); err != nil {
				return nil, nil, err
			}
			if value.MatchMissing, err = takeObservabilityControlBool(control, "matchMissing", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Required, err = takeObservabilityControlBool(control, "required", controlPath); err != nil {
				return nil, nil, err
			}
			if value.ApplicationMode, err = takeObservabilityControlString(control, "applicationMode", controlPath); err != nil {
				return nil, nil, err
			}
			model.PinnedFilter = append(model.PinnedFilter, *value)
		case "FilterSet":
			if variableName.ValueString() != observabilityFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			if seenSingleton[observabilityFilterSetVariableName] {
				return nil, nil, fmt.Errorf("%s duplicates the canonical FilterSet control", controlPath)
			}
			seenSingleton[observabilityFilterSetVariableName] = true
			value := &observabilityFilterSetControlModel{}
			if value.Label, err = takeObservabilityControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeObservabilityControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeObservabilityControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			var filterLeftovers []string
			if value.Filter, filterLeftovers, err = takeObservabilityFilterSetEntries(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.FilterSet = value
			leftovers = append(leftovers, filterLeftovers...)
		default:
			leftovers = append(leftovers, controlPath)
			continue
		}
		leftovers = append(leftovers, observabilityLeftovers(controlPath, control)...)
	}
	leftovers = append(leftovers, observabilityLeftovers("controlBar", bar)...)
	return model, leftovers, nil
}

func takeObservabilityControlString(object map[string]any, key, path string) (types.String, error) {
	raw, ok := object[key]
	if !ok {
		return types.StringNull(), nil
	}
	value, ok := raw.(string)
	if !ok {
		return types.StringNull(), fmt.Errorf("%s.%s is %v (%T) rather than a string", path, key, raw, raw)
	}
	delete(object, key)
	return types.StringValue(value), nil
}

func takeObservabilityControlBool(object map[string]any, key, path string) (types.Bool, error) {
	raw, ok := object[key]
	if !ok {
		return types.BoolNull(), nil
	}
	value, ok := raw.(bool)
	if !ok {
		return types.BoolNull(), fmt.Errorf("%s.%s is %v (%T) rather than a boolean", path, key, raw, raw)
	}
	delete(object, key)
	return types.BoolValue(value), nil
}

func takeObservabilityControlInt64(object map[string]any, key, path string) (types.Int64, error) {
	raw, ok := object[key]
	if !ok {
		return types.Int64Null(), nil
	}
	var value int64
	switch number := raw.(type) {
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number || number > math.MaxInt64 || number < math.MinInt64 {
			return types.Int64Null(), fmt.Errorf("%s.%s is %v rather than an integer", path, key, raw)
		}
		value = int64(number)
	default:
		return types.Int64Null(), fmt.Errorf("%s.%s is %v (%T) rather than an integer", path, key, raw, raw)
	}
	delete(object, key)
	return types.Int64Value(value), nil
}

func takeObservabilityControlStringList(object map[string]any, key, path string) ([]types.String, error) {
	raw, ok := object[key]
	if !ok {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.%s is %v (%T) rather than a list", path, key, raw, raw)
	}
	result := make([]types.String, len(values))
	for i, rawValue := range values {
		value, ok := rawValue.(string)
		if !ok {
			return nil, fmt.Errorf("%s.%s.%d is %v (%T) rather than a string", path, key, i, rawValue, rawValue)
		}
		result[i] = types.StringValue(value)
	}
	delete(object, key)
	return result, nil
}

func takeObservabilityFilterSetEntries(object map[string]any, key, path string) ([]observabilityFilterSetEntryModel, []string, error) {
	raw, ok := object[key]
	if !ok {
		return nil, nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s.%s is %v (%T) rather than a list", path, key, raw, raw)
	}
	result := make([]observabilityFilterSetEntryModel, len(values))
	var leftovers []string
	for i, rawValue := range values {
		entryPath := fmt.Sprintf("%s.%s.%d", path, key, i)
		entry, ok := rawValue.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%s is %T rather than an object", entryPath, rawValue)
		}
		filter := observabilityFilterSetEntryModel{}
		var err error
		if filter.Key, err = takeObservabilityControlString(entry, "key", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Key.IsNull() || filter.Key.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.key must be a non-empty string", entryPath)
		}
		if filter.Values, err = takeObservabilityControlStringList(entry, "values", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Values == nil {
			return nil, nil, fmt.Errorf("%s.values must be a list", entryPath)
		}
		if filter.Negated, err = takeObservabilityControlBool(entry, "negated", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Disabled, err = takeObservabilityControlBool(entry, "disabled", entryPath); err != nil {
			return nil, nil, err
		}
		// The filter entry itself is an independently modeled object, so its
		// unknown fields are reported at the precise nested path.
		result[i] = filter
		leftovers = append(leftovers, observabilityLeftovers(entryPath, entry)...)
	}
	delete(object, key)
	return result, leftovers, nil
}
