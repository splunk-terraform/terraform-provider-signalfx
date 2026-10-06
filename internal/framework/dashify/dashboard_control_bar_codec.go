// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	dashifyTimeRangeVariableName = "TIME"
	dashifyDensityVariableName   = "DENSITY"
	dashifyFilterSetVariableName = "FILTERS"
)

// encodes a Terraform control_bar model into the Dashify controls
// list; parseDashifyControlBar performs the inverse decode.
func buildDashifyControlBar(model *dashifyControlBarModel) map[string]any {
	controls := make([]any, 0)
	if model.TimeRange != nil {
		control := map[string]any{
			"type":         "TimeRange",
			"variableName": dashifyTimeRangeVariableName,
		}
		setDashifyControlString(control, "label", model.TimeRange.Label)
		setDashifyControlString(control, "description", model.TimeRange.Description)
		setDashifyControlBool(control, "hidden", model.TimeRange.Hidden)
		setDashifyControlString(control, "defaultVariableValue", model.TimeRange.DefaultVariableValue)
		controls = append(controls, control)
	}
	if model.Density != nil {
		control := map[string]any{
			"type":         "Density",
			"variableName": dashifyDensityVariableName,
		}
		setDashifyControlString(control, "label", model.Density.Label)
		setDashifyControlString(control, "description", model.Density.Description)
		setDashifyControlBool(control, "hidden", model.Density.Hidden)
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
		setDashifyControlString(control, "label", filter.Label)
		setDashifyControlString(control, "description", filter.Description)
		setDashifyControlBool(control, "hidden", filter.Hidden)
		setDashifyControlString(control, "key", filter.Key)
		setDashifyControlStringList(control, "defaultVariableValue", filter.DefaultVariableValue)
		setDashifyControlStringList(control, "preferablySuggestedValues", filter.SuggestedValues)
		setDashifyControlBool(control, "onlySuggestPreferredValues", filter.OnlySuggestPreferredValues)
		setDashifyControlBool(control, "matchMissing", filter.MatchMissing)
		setDashifyControlBool(control, "required", filter.Required)
		setDashifyControlString(control, "applicationMode", filter.ApplicationMode)
		controls = append(controls, control)
	}
	if model.FilterSet != nil {
		control := map[string]any{
			"type":         "FilterSet",
			"variableName": dashifyFilterSetVariableName,
		}
		setDashifyControlString(control, "label", model.FilterSet.Label)
		setDashifyControlString(control, "description", model.FilterSet.Description)
		setDashifyControlBool(control, "hidden", model.FilterSet.Hidden)
		entries := make([]any, len(model.FilterSet.Filter))
		for i, filter := range model.FilterSet.Filter {
			entry := map[string]any{
				"key":    filter.Key.ValueString(),
				"values": dashifyControlStringsToAny(filter.Values),
			}
			setDashifyControlBool(entry, "negated", filter.Negated)
			setDashifyControlBool(entry, "disabled", filter.Disabled)
			setDashifyControlBool(entry, "matchMissing", filter.MatchMissing)
			entries[i] = entry
		}
		control["defaultVariableValue"] = entries
		controls = append(controls, control)
	}
	return map[string]any{"controls": controls}
}

func setDashifyControlString(destination map[string]any, key string, value types.String) {
	if !value.IsNull() && !value.IsUnknown() {
		destination[key] = value.ValueString()
	}
}

func setDashifyControlBool(destination map[string]any, key string, value types.Bool) {
	if !value.IsNull() && !value.IsUnknown() {
		destination[key] = value.ValueBool()
	}
}

func setDashifyControlStringList(destination map[string]any, key string, values []types.String) {
	if values == nil {
		return
	}
	destination[key] = dashifyControlStringsToAny(values)
}

func dashifyControlStringsToAny(values []types.String) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
}

// decodes a Dashify controls list into a Terraform control_bar model,
// the inverse of buildDashifyControlBar. TimeRange, Density, and FilterSet
// are recognized only by their fixed variableName; a control sharing that
// type but a different variableName is left as a leftover instead of
// overwriting the canonical one.
func parseDashifyControlBar(spec map[string]any) (*dashifyControlBarModel, []string, error) {
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
	if len(controls) == 0 {
		return nil, dashifyLeftovers("controlBar", bar), nil
	}

	model := &dashifyControlBarModel{}
	var leftovers []string
	for i, rawControl := range controls {
		controlPath := fmt.Sprintf("controlBar.controls.%d", i)
		control, ok := rawControl.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%s is %T rather than an object", controlPath, rawControl)
		}
		controlType, err := takeDashifyControlString(control, "type", controlPath)
		if err != nil {
			return nil, nil, err
		}
		variableName, err := takeDashifyControlString(control, "variableName", controlPath)
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
			if variableName.ValueString() != dashifyTimeRangeVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := &dashifyTimeRangeControlModel{}
			if value.Label, err = takeDashifyControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeDashifyControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeDashifyControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeDashifyControlString(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.TimeRange = value
		case "Density":
			if variableName.ValueString() != dashifyDensityVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := &dashifyDensityControlModel{}
			if value.Label, err = takeDashifyControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeDashifyControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeDashifyControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeDashifyControlInt64(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.Density = value
		case "PinnedFilter":
			if variableName.ValueString() == dashifyTimeRangeVariableName || variableName.ValueString() == dashifyDensityVariableName || variableName.ValueString() == dashifyFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := &dashifyPinnedFilterControlModel{VariableName: variableName}
			if value.Label, err = takeDashifyControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeDashifyControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeDashifyControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Key, err = takeDashifyControlString(control, "key", controlPath); err != nil {
				return nil, nil, err
			}
			if value.DefaultVariableValue, err = takeDashifyControlStringList(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			if value.SuggestedValues, err = takeDashifyControlStringList(control, "preferablySuggestedValues", controlPath); err != nil {
				return nil, nil, err
			}
			if value.OnlySuggestPreferredValues, err = takeDashifyControlBool(control, "onlySuggestPreferredValues", controlPath); err != nil {
				return nil, nil, err
			}
			if value.MatchMissing, err = takeDashifyControlBool(control, "matchMissing", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Required, err = takeDashifyControlBool(control, "required", controlPath); err != nil {
				return nil, nil, err
			}
			if value.ApplicationMode, err = takeDashifyControlString(control, "applicationMode", controlPath); err != nil {
				return nil, nil, err
			}
			model.PinnedFilter = append(model.PinnedFilter, *value)
		case "FilterSet":
			if variableName.ValueString() != dashifyFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := &dashifyFilterSetControlModel{}
			if value.Label, err = takeDashifyControlString(control, "label", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Description, err = takeDashifyControlString(control, "description", controlPath); err != nil {
				return nil, nil, err
			}
			if value.Hidden, err = takeDashifyControlBool(control, "hidden", controlPath); err != nil {
				return nil, nil, err
			}
			var filterLeftovers []string
			if value.Filter, filterLeftovers, err = takeDashifyFilterSetEntries(control, "defaultVariableValue", controlPath); err != nil {
				return nil, nil, err
			}
			model.FilterSet = value
			leftovers = append(leftovers, filterLeftovers...)
		default:
			leftovers = append(leftovers, controlPath)
			continue
		}
		leftovers = append(leftovers, dashifyLeftovers(controlPath, control)...)
	}
	leftovers = append(leftovers, dashifyLeftovers("controlBar", bar)...)
	return model, leftovers, nil
}

func takeDashifyControlString(object map[string]any, key, path string) (types.String, error) {
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

func takeDashifyControlBool(object map[string]any, key, path string) (types.Bool, error) {
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

func takeDashifyControlInt64(object map[string]any, key, path string) (types.Int64, error) {
	raw, ok := object[key]
	if !ok {
		return types.Int64Null(), nil
	}
	value, valid := dashifyExactInt64(raw)
	if !valid {
		return types.Int64Null(), fmt.Errorf("%s.%s is %v (%T) rather than an integer", path, key, raw, raw)
	}
	delete(object, key)
	return types.Int64Value(value), nil
}

func takeDashifyControlStringList(object map[string]any, key, path string) ([]types.String, error) {
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

func takeDashifyFilterSetEntries(object map[string]any, key, path string) ([]dashifyFilterSetEntryModel, []string, error) {
	raw, ok := object[key]
	if !ok {
		return nil, nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s.%s is %v (%T) rather than a list", path, key, raw, raw)
	}
	result := make([]dashifyFilterSetEntryModel, len(values))
	var leftovers []string
	for i, rawValue := range values {
		entryPath := fmt.Sprintf("%s.%s.%d", path, key, i)
		entry, ok := rawValue.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%s is %T rather than an object", entryPath, rawValue)
		}
		filter := dashifyFilterSetEntryModel{}
		var err error
		if filter.Key, err = takeDashifyControlString(entry, "key", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Key.IsNull() || filter.Key.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.key must be a non-empty string", entryPath)
		}
		if filter.Values, err = takeDashifyControlStringList(entry, "values", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Values == nil {
			return nil, nil, fmt.Errorf("%s.values must be a list", entryPath)
		}
		if filter.Negated, err = takeDashifyControlBool(entry, "negated", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.Disabled, err = takeDashifyControlBool(entry, "disabled", entryPath); err != nil {
			return nil, nil, err
		}
		if filter.MatchMissing, err = takeDashifyControlBool(entry, "matchMissing", entryPath); err != nil {
			return nil, nil, err
		}
		// The filter entry itself is an independently modeled object, so its
		// unknown fields are reported at the precise nested path.
		result[i] = filter
		leftovers = append(leftovers, dashifyLeftovers(entryPath, entry)...)
	}
	delete(object, key)
	return result, leftovers, nil
}
