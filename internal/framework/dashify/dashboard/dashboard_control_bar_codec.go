// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	dashifyTimeRangeVariableName = "TIME"
	dashifyDensityVariableName   = "DENSITY"
	dashifyFilterSetVariableName = "FILTERS"
)

// encodes a Terraform control_bar model into the Dashify controls list
// parseDashifyControlBar performs the inverse decode.
func buildDashifyControlBar(model *dashifyControlBarModel) map[string]any {
	controls := make([]map[string]any, 0)
	if model.TimeRange != nil {
		control := map[string]any{
			"type":         "TimeRange",
			"variableName": dashifyTimeRangeVariableName,
		}
		setDashifyControlString(control, "label", model.TimeRange.Label)
		setDashifyControlString(control, "description", model.TimeRange.Description)
		setDashifyControlBool(control, "hidden", model.TimeRange.Hidden)
		setDashifyControlString(control, "defaultVariableValue", model.TimeRange.Default)
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
		if !model.Density.Default.IsNull() && !model.Density.Default.IsUnknown() {
			control["defaultVariableValue"] = model.Density.Default.ValueInt64()
		}
		controls = append(controls, control)
	}
	for _, filter := range model.PinnedFilter {
		control := map[string]any{
			"type":         "PinnedFilter",
			"variableName": filter.effectiveVariableName().ValueString(),
		}
		setDashifyControlString(control, "label", filter.Label)
		setDashifyControlString(control, "description", filter.Description)
		setDashifyControlBool(control, "hidden", filter.Hidden)
		setDashifyControlString(control, "key", filter.Property)
		setDashifyControlStringList(control, "defaultVariableValue", filter.Default)
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
				"key":    filter.Property.ValueString(),
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

	// The UI dedupes controls by variableName with last-occurrence-wins
	// Pre-scan to find each variableName's last index so the main loop
	// can match that behavior.
	latestVariableIndex := map[string]int{}
	for i, rawControl := range controls {
		control, ok := rawControl.(map[string]any)
		if !ok {
			continue
		}
		name, ok := control["variableName"].(string)
		if !ok || name == "" {
			continue
		}
		latestVariableIndex[name] = i
	}

	model := &dashifyControlBarModel{}
	var leftovers []string
	for i, rawControl := range controls {
		controlPath := fmt.Sprintf("controlBar.controls.%d", i)
		control, ok := rawControl.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%s is %T rather than an object", controlPath, rawControl)
		}
		reader := dashifyControlReader{object: control, path: controlPath}
		controlType := reader.string("type")
		variableName := reader.string("variableName")
		if reader.err != nil {
			return nil, nil, reader.err
		}
		if variableName.IsNull() || variableName.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.variableName must be a non-empty string", controlPath)
		}
		if controlType.ValueString() == "" {
			return nil, nil, fmt.Errorf("%s.type must be a non-empty string", controlPath)
		}

		if latestVariableIndex[variableName.ValueString()] != i {
			leftovers = append(leftovers, controlPath)
			continue
		}

		switch controlType.ValueString() {
		case "TimeRange":
			if variableName.ValueString() != dashifyTimeRangeVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			model.TimeRange = &dashifyTimeRangeControlModel{
				Label:       reader.optionalString("label"),
				Description: reader.optionalString("description"),
				Hidden:      reader.bool("hidden"),
				Default:     reader.string("defaultVariableValue"),
			}
		case "Density":
			if variableName.ValueString() != dashifyDensityVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			model.Density = &dashifyDensityControlModel{
				Label:       reader.optionalString("label"),
				Description: reader.optionalString("description"),
				Hidden:      reader.bool("hidden"),
				Default:     reader.int64("defaultVariableValue"),
			}
		case "PinnedFilter":
			if variableName.ValueString() == dashifyTimeRangeVariableName || variableName.ValueString() == dashifyDensityVariableName || variableName.ValueString() == dashifyFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := dashifyPinnedFilterControlModel{
				VariableName:               variableName,
				Label:                      reader.optionalString("label"),
				Description:                reader.optionalString("description"),
				Hidden:                     reader.bool("hidden"),
				Property:                   reader.string("key"),
				Default:                    reader.stringList("defaultVariableValue"),
				SuggestedValues:            reader.stringList("preferablySuggestedValues"),
				OnlySuggestPreferredValues: reader.bool("onlySuggestPreferredValues"),
				MatchMissing:               reader.bool("matchMissing"),
				Required:                   reader.bool("required"),
				ApplicationMode:            reader.string("applicationMode"),
			}
			if value.Property.IsNull() {
				value.Property = variableName
			}
			model.PinnedFilter = append(model.PinnedFilter, value)
		case "FilterSet":
			if variableName.ValueString() != dashifyFilterSetVariableName {
				leftovers = append(leftovers, controlPath)
				continue
			}
			value := &dashifyFilterSetControlModel{
				Label:       reader.optionalString("label"),
				Description: reader.optionalString("description"),
				Hidden:      reader.bool("hidden"),
			}
			filters, filterLeftovers := reader.filterSetEntries("defaultVariableValue")
			value.Filter = filters
			model.FilterSet = value
			leftovers = append(leftovers, filterLeftovers...)
		default:
			leftovers = append(leftovers, controlPath)
			continue
		}
		if reader.err != nil {
			return nil, nil, reader.err
		}
		leftovers = append(leftovers, dashifyLeftovers(controlPath, control)...)
	}
	leftovers = append(leftovers, dashifyLeftovers("controlBar", bar)...)
	return model, leftovers, nil
}
