// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dashifyControlBarBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Optional controls applied to the dashboard and its charts.",
		Blocks: map[string]schema.Block{
			"time_range": schema.SingleNestedBlock{
				Description: "The dashboard time-range control. Its reserved variable name is always TIME.",
				Attributes: func() map[string]schema.Attribute {
					attributes := dashifyCommonControlAttributes("time-range")
					attributes["default_variable_value"] = schema.StringAttribute{
						Optional:    true,
						Description: "Default time-range value, such as -15m, -PT15M, or an absolute time range.",
						Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
					}
					return attributes
				}(),
			},
			"density": schema.SingleNestedBlock{
				Description: "The chart density control. Its reserved variable name is always DENSITY.",
				Attributes: func() map[string]schema.Attribute {
					attributes := dashifyCommonControlAttributes("density")
					attributes["default_variable_value"] = schema.Int64Attribute{
						Optional:    true,
						Description: "Default chart resolution in seconds. Supported values are 30, 60, 120, and 240.",
						Validators:  []validator.Int64{int64validator.OneOf(30, 60, 120, 240)},
					}
					return attributes
				}(),
			},
			"pinned_filter": schema.ListNestedBlock{
				Description: "An ordered filter control pinned to the dashboard bar.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"variable_name": schema.StringAttribute{
							Required:    true,
							Description: "Unique control identity. TIME, DENSITY, and FILTERS are reserved.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"label":       optionalControlString("Control label."),
						"description": optionalControlString("Control description."),
						"hidden":      schema.BoolAttribute{Optional: true, Description: "Whether to hide this control."},
						"key": schema.StringAttribute{
							Optional:    true,
							Description: "Property to filter. Defaults to variable_name.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"default_variable_value": optionalStringList("Default selected filter values."),
						"suggested_values":       optionalStringList("Preferred filter suggestions."),
						"only_suggest_preferred_values": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether only preferred suggestions should be offered.",
						},
						"match_missing": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether values missing the property should match.",
						},
						"required": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether a visible filter must have a selected value before charts render.",
						},
						"application_mode": schema.StringAttribute{
							Optional:    true,
							Description: "How the filter is applied: add, override, or ignore.",
							Validators:  []validator.String{stringvalidator.OneOf("add", "override", "ignore")},
						},
					},
				},
			},
			"filter_set": schema.SingleNestedBlock{
				Description: "The ad-hoc filter picker and its optional default filters. Its reserved variable name is always FILTERS.",
				Attributes:  dashifyCommonControlAttributes("filter-set"),
				Blocks: map[string]schema.Block{
					"filter": schema.ListNestedBlock{
						Description: "Ordered default filters for the filter-set control. Empty values are a no-op.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:    true,
									Description: "Property to filter.",
									Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
								},
								"values": schema.ListAttribute{
									Required:    true,
									ElementType: types.StringType,
									Description: "Values for the filter. An empty list has no effect.",
									Validators:  nonEmptyStringListValidators(),
								},
								"negated": schema.BoolAttribute{
									Optional:    true,
									Description: "Whether to negate this filter.",
								},
								"disabled": schema.BoolAttribute{
									Optional:    true,
									Description: "Whether to disable this default filter.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func dashifyCommonControlAttributes(controlName string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"label":       optionalControlString(fmt.Sprintf("The %s control label.", controlName)),
		"description": optionalControlString(fmt.Sprintf("The %s control description.", controlName)),
		"hidden":      schema.BoolAttribute{Optional: true, Description: fmt.Sprintf("Whether to hide the %s control.", controlName)},
	}
}

func optionalControlString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: description,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
	}
}

func optionalStringList(description string) schema.ListAttribute {
	return schema.ListAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: description,
		Validators:  nonEmptyStringListValidators(),
	}
}

func validateDashifyControlBar(resp *resource.ValidateConfigResponse, controlBarPath path.Path, model *dashifyControlBarModel) {
	if model == nil {
		return
	}
	if model.TimeRange == nil && model.Density == nil && len(model.PinnedFilter) == 0 && model.FilterSet == nil {
		resp.Diagnostics.AddAttributeError(controlBarPath, "Empty control bar", "control_bar must contain at least one time_range, density, pinned_filter, or filter_set block")
		return
	}

	seen := make(map[string]struct{}, len(model.PinnedFilter))
	for i, pinned := range model.PinnedFilter {
		namePath := controlBarPath.AtName("pinned_filter").AtListIndex(i).AtName("variable_name")
		if pinned.VariableName.IsNull() || pinned.VariableName.IsUnknown() || pinned.VariableName.ValueString() == "" {
			continue
		}
		name := pinned.VariableName.ValueString()
		if name == dashifyTimeRangeVariableName || name == dashifyDensityVariableName || name == dashifyFilterSetVariableName {
			resp.Diagnostics.AddAttributeError(namePath, "Reserved control variable name", fmt.Sprintf("%q is reserved for the dashboard singleton controls", name))
			continue
		}
		if _, exists := seen[name]; exists {
			resp.Diagnostics.AddAttributeError(namePath, "Duplicate control variable name", fmt.Sprintf("another pinned_filter already uses variable_name %q", name))
			continue
		}
		seen[name] = struct{}{}
	}
}
