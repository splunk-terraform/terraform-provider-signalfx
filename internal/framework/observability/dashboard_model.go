// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import "github.com/hashicorp/terraform-plugin-framework/types"

// The transport models exactly match the schema available at each nesting
// level. Terraform Plugin Framework requires this one-to-one correspondence.
type observabilityDashboardModel struct {
	ID         types.String                     `tfsdk:"id"`
	Title      types.String                     `tfsdk:"title"`
	ControlBar *dashifyControlBarModel          `tfsdk:"control_bar"`
	Layout     *dashifyLayoutOptionsModel       `tfsdk:"layout"`
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
	Key      types.String   `tfsdk:"key"`
	Values   []types.String `tfsdk:"values"`
	Negated  types.Bool     `tfsdk:"negated"`
	Disabled types.Bool     `tfsdk:"disabled"`
}

type dashifyLayoutModel struct {
	Absolute  types.Bool   `tfsdk:"absolute"`
	Width     types.String `tfsdk:"width"`
	Height    types.String `tfsdk:"height"`
	MinWidth  types.String `tfsdk:"min_width"`
	MaxWidth  types.String `tfsdk:"max_width"`
	MinHeight types.String `tfsdk:"min_height"`
	MaxHeight types.String `tfsdk:"max_height"`
	X         types.String `tfsdk:"x"`
	Y         types.String `tfsdk:"y"`
}

type dashifyLayoutOptionsModel struct {
	Gap      types.Float64               `tfsdk:"gap"`
	Step     types.Float64               `tfsdk:"step"`
	Defaults *dashifyLayoutDefaultsModel `tfsdk:"defaults"`
}

type dashifyLayoutDefaultsModel struct {
	Absolute  types.Bool   `tfsdk:"absolute"`
	Width     types.String `tfsdk:"width"`
	Height    types.String `tfsdk:"height"`
	MinWidth  types.String `tfsdk:"min_width"`
	MaxWidth  types.String `tfsdk:"max_width"`
	MinHeight types.String `tfsdk:"min_height"`
	MaxHeight types.String `tfsdk:"max_height"`
}

type dashifyTemplateModel struct {
	TemplateID types.String `tfsdk:"template_id"`
	Content    types.String `tfsdk:"content"`
}

// TODO(charts): Generate the level-specific container chart fields from the
// external Dashify schemas. The generated models should expose inline blocks
// such as metrics_single_value and metrics_timeseries at every container level.
type dashifyDashboardContainerModel struct {
	Layout   *dashifyLayoutModel   `tfsdk:"layout"`
	Template *dashifyTemplateModel `tfsdk:"template"`
	Section  *dashifySectionModel  `tfsdk:"section"`
	Group    *dashifyGroupModel    `tfsdk:"group"`
}

type dashifySectionModel struct {
	Title       types.String                   `tfsdk:"title"`
	Collapse    types.Bool                     `tfsdk:"collapse"`
	Collapsible types.Bool                     `tfsdk:"collapsible"`
	Layout      *dashifyLayoutOptionsModel     `tfsdk:"layout"`
	Container   []dashifySectionContainerModel `tfsdk:"container"`
}

type dashifySectionContainerModel struct {
	Layout   *dashifyLayoutModel   `tfsdk:"layout"`
	Template *dashifyTemplateModel `tfsdk:"template"`
	Group    *dashifyGroupModel    `tfsdk:"group"`
}

type dashifyGroupModel struct {
	Title      types.String                 `tfsdk:"title"`
	Headerless types.Bool                   `tfsdk:"headerless"`
	Layout     *dashifyLayoutOptionsModel   `tfsdk:"layout"`
	Container  []dashifyGroupContainerModel `tfsdk:"container"`
}

type dashifyGroupContainerModel struct {
	Layout   *dashifyLayoutModel   `tfsdk:"layout"`
	Template *dashifyTemplateModel `tfsdk:"template"`
}

// dashifyContainer is the shared semantic model used after decoding and
// before encoding the level-specific Terraform transport models.
type dashifyContainer struct {
	Layout   *dashifyLayoutModel
	Template *dashifyTemplateModel
	Section  *dashifySection
	Group    *dashifyGroup
	// TODO(charts): Carry the generated chart-content abstraction through this
	// shared model so metrics_single_value, metrics_timeseries, and later schema
	// additions do not require hand-written fields here.
}

type dashifySection struct {
	Title       types.String
	Collapse    types.Bool
	Collapsible types.Bool
	Layout      *dashifyLayoutOptionsModel
	Container   []dashifyContainer
}

type dashifyGroup struct {
	Title      types.String
	Headerless types.Bool
	Layout     *dashifyLayoutOptionsModel
	Container  []dashifyContainer
}

type dashifyContainerLevel uint8

const (
	dashifyDashboardContainerLevel dashifyContainerLevel = iota
	dashifySectionContainerLevel
	dashifyGroupContainerLevel
)

func dashifyContainersFromDashboardModels(models []dashifyDashboardContainerModel) []dashifyContainer {
	containers := make([]dashifyContainer, len(models))
	for i, model := range models {
		container := dashifyContainer{Layout: model.Layout, Template: model.Template}
		if model.Section != nil {
			container.Section = &dashifySection{
				Title:       model.Section.Title,
				Collapse:    model.Section.Collapse,
				Collapsible: model.Section.Collapsible,
				Layout:      model.Section.Layout,
				Container:   dashifyContainersFromSectionModels(model.Section.Container),
			}
		}
		if model.Group != nil {
			container.Group = dashifyGroupFromModel(model.Group)
		}
		containers[i] = container
	}
	return containers
}

func dashifyContainersFromSectionModels(models []dashifySectionContainerModel) []dashifyContainer {
	containers := make([]dashifyContainer, len(models))
	for i, model := range models {
		container := dashifyContainer{Layout: model.Layout, Template: model.Template}
		if model.Group != nil {
			container.Group = dashifyGroupFromModel(model.Group)
		}
		containers[i] = container
	}
	return containers
}

func dashifyGroupFromModel(model *dashifyGroupModel) *dashifyGroup {
	return &dashifyGroup{
		Title:      model.Title,
		Headerless: model.Headerless,
		Layout:     model.Layout,
		Container:  dashifyContainersFromGroupModels(model.Container),
	}
}

func dashifyContainersFromGroupModels(models []dashifyGroupContainerModel) []dashifyContainer {
	containers := make([]dashifyContainer, len(models))
	for i, model := range models {
		containers[i] = dashifyContainer{Layout: model.Layout, Template: model.Template}
	}
	return containers
}

func dashifyDashboardModelsFromContainers(containers []dashifyContainer) []dashifyDashboardContainerModel {
	models := make([]dashifyDashboardContainerModel, len(containers))
	for i, container := range containers {
		model := dashifyDashboardContainerModel{Layout: container.Layout, Template: container.Template}
		if container.Section != nil {
			model.Section = &dashifySectionModel{
				Title:       container.Section.Title,
				Collapse:    container.Section.Collapse,
				Collapsible: container.Section.Collapsible,
				Layout:      container.Section.Layout,
				Container:   dashifySectionModelsFromContainers(container.Section.Container),
			}
		}
		if container.Group != nil {
			model.Group = dashifyGroupModelFromGroup(container.Group)
		}
		models[i] = model
	}
	return models
}

func dashifySectionModelsFromContainers(containers []dashifyContainer) []dashifySectionContainerModel {
	models := make([]dashifySectionContainerModel, len(containers))
	for i, container := range containers {
		model := dashifySectionContainerModel{Layout: container.Layout, Template: container.Template}
		if container.Group != nil {
			model.Group = dashifyGroupModelFromGroup(container.Group)
		}
		models[i] = model
	}
	return models
}

func dashifyGroupModelFromGroup(group *dashifyGroup) *dashifyGroupModel {
	return &dashifyGroupModel{
		Title:      group.Title,
		Headerless: group.Headerless,
		Layout:     group.Layout,
		Container:  dashifyGroupModelsFromContainers(group.Container),
	}
}

func dashifyGroupModelsFromContainers(containers []dashifyContainer) []dashifyGroupContainerModel {
	models := make([]dashifyGroupContainerModel, len(containers))
	for i, container := range containers {
		models[i] = dashifyGroupContainerModel{Layout: container.Layout, Template: container.Template}
	}
	return models
}
