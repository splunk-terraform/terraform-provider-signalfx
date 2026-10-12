// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type observabilityDashboardModel struct {
	ID         types.String                     `tfsdk:"id"`
	Title      types.String                     `tfsdk:"title"`
	ControlBar *dashifyControlBarModel          `tfsdk:"control_bar"`
	Container  []dashifyDashboardContainerModel `tfsdk:"container"`
}

type dashifyControlBarModel struct {
	TimeRange    *dashifyTimeRangeControlModel     `tfsdk:"time_range"`
	Density      *dashifyDensityControlModel       `tfsdk:"density"`
	PinnedFilter []dashifyPinnedFilterControlModel `tfsdk:"pinned_filter"`
	FilterSet    *dashifyFilterSetControlModel     `tfsdk:"filter_set"`
}

type dashifyTimeRangeControlModel struct {
	Label                types.String `tfsdk:"label"`
	Description          types.String `tfsdk:"description"`
	Hidden               types.Bool   `tfsdk:"hidden"`
	DefaultVariableValue types.String `tfsdk:"default_variable_value"`
}

type dashifyDensityControlModel struct {
	Label                types.String `tfsdk:"label"`
	Description          types.String `tfsdk:"description"`
	Hidden               types.Bool   `tfsdk:"hidden"`
	DefaultVariableValue types.Int64  `tfsdk:"default_variable_value"`
}

type dashifyPinnedFilterControlModel struct {
	VariableName               types.String   `tfsdk:"variable_name"`
	Label                      types.String   `tfsdk:"label"`
	Description                types.String   `tfsdk:"description"`
	Hidden                     types.Bool     `tfsdk:"hidden"`
	Key                        types.String   `tfsdk:"key"`
	DefaultVariableValue       []types.String `tfsdk:"default_variable_value"`
	SuggestedValues            []types.String `tfsdk:"suggested_values"`
	OnlySuggestPreferredValues types.Bool     `tfsdk:"only_suggest_preferred_values"`
	MatchMissing               types.Bool     `tfsdk:"match_missing"`
	Required                   types.Bool     `tfsdk:"required"`
	ApplicationMode            types.String   `tfsdk:"application_mode"`
}

type dashifyFilterSetControlModel struct {
	Label       types.String                 `tfsdk:"label"`
	Description types.String                 `tfsdk:"description"`
	Hidden      types.Bool                   `tfsdk:"hidden"`
	Filter      []dashifyFilterSetEntryModel `tfsdk:"filter"`
}

type dashifyFilterSetEntryModel struct {
	Key          types.String   `tfsdk:"key"`
	Values       []types.String `tfsdk:"values"`
	Negated      types.Bool     `tfsdk:"negated"`
	Disabled     types.Bool     `tfsdk:"disabled"`
	MatchMissing types.Bool     `tfsdk:"match_missing"`
}

type dashifyLayoutModel struct {
	Width     types.String `tfsdk:"width"`
	Height    types.String `tfsdk:"height"`
	MinWidth  types.String `tfsdk:"min_width"`
	MinHeight types.String `tfsdk:"min_height"`
}

type dashifyDashboardContainerModel struct {
	Layout          *dashifyLayoutModel  `tfsdk:"layout"`
	TemplateID      types.String         `tfsdk:"template_id"`
	InlineContent jsontypes.Normalized `tfsdk:"inline_content"`
}
