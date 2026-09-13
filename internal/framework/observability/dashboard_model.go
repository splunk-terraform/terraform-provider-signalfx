// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import "github.com/hashicorp/terraform-plugin-framework/types"

// The transport models exactly match the schema available at each nesting
// level. Terraform Plugin Framework requires this one-to-one correspondence.
type observabilityDashboardModel struct {
	ID         types.String                           `tfsdk:"id"`
	Title      types.String                           `tfsdk:"title"`
	ControlBar *observabilityControlBarModel          `tfsdk:"control_bar"`
	Layout     *observabilityLayoutOptionsModel       `tfsdk:"layout"`
	Container  []observabilityDashboardContainerModel `tfsdk:"container"`
}

type observabilityControlBarModel struct {
	TimeRange    *observabilityTimeRangeControlModel     `tfsdk:"time_range"`
	Density      *observabilityDensityControlModel       `tfsdk:"density"`
	PinnedFilter []observabilityPinnedFilterControlModel `tfsdk:"pinned_filter"`
	FilterSet    *observabilityFilterSetControlModel     `tfsdk:"filter_set"`
}

type observabilityTimeRangeControlModel struct {
	Label                types.String `tfsdk:"label"`
	Description          types.String `tfsdk:"description"`
	Hidden               types.Bool   `tfsdk:"hidden"`
	DefaultVariableValue types.String `tfsdk:"default_variable_value"`
}

type observabilityDensityControlModel struct {
	Label                types.String `tfsdk:"label"`
	Description          types.String `tfsdk:"description"`
	Hidden               types.Bool   `tfsdk:"hidden"`
	DefaultVariableValue types.Int64  `tfsdk:"default_variable_value"`
}

type observabilityPinnedFilterControlModel struct {
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

type observabilityFilterSetControlModel struct {
	Label       types.String                       `tfsdk:"label"`
	Description types.String                       `tfsdk:"description"`
	Hidden      types.Bool                         `tfsdk:"hidden"`
	Filter      []observabilityFilterSetEntryModel `tfsdk:"filter"`
}

type observabilityFilterSetEntryModel struct {
	Key      types.String   `tfsdk:"key"`
	Values   []types.String `tfsdk:"values"`
	Negated  types.Bool     `tfsdk:"negated"`
	Disabled types.Bool     `tfsdk:"disabled"`
}

type observabilityLayoutModel struct {
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

type observabilityLayoutOptionsModel struct {
	Gap      types.Float64                     `tfsdk:"gap"`
	Step     types.Float64                     `tfsdk:"step"`
	Defaults *observabilityLayoutDefaultsModel `tfsdk:"defaults"`
}

type observabilityLayoutDefaultsModel struct {
	Absolute  types.Bool   `tfsdk:"absolute"`
	Width     types.String `tfsdk:"width"`
	Height    types.String `tfsdk:"height"`
	MinWidth  types.String `tfsdk:"min_width"`
	MaxWidth  types.String `tfsdk:"max_width"`
	MinHeight types.String `tfsdk:"min_height"`
	MaxHeight types.String `tfsdk:"max_height"`
}

type observabilityDashboardTemplateModel struct {
	TemplateID types.String `tfsdk:"template_id"`
	Content    types.String `tfsdk:"content"`
}

// TODO(charts): Generate the level-specific container chart fields from the
// external Dashify schemas. The generated models should expose inline blocks
// such as metrics_single_value and metrics_timeseries at every container level.
type observabilityDashboardContainerModel struct {
	Layout   *observabilityLayoutModel            `tfsdk:"layout"`
	Template *observabilityDashboardTemplateModel `tfsdk:"template"`
	Section  *observabilitySectionModel           `tfsdk:"section"`
	Group    *observabilityGroupModel             `tfsdk:"group"`
}

type observabilitySectionModel struct {
	Title       types.String                         `tfsdk:"title"`
	Collapse    types.Bool                           `tfsdk:"collapse"`
	Collapsible types.Bool                           `tfsdk:"collapsible"`
	Layout      *observabilityLayoutOptionsModel     `tfsdk:"layout"`
	Container   []observabilitySectionContainerModel `tfsdk:"container"`
}

type observabilitySectionContainerModel struct {
	Layout   *observabilityLayoutModel            `tfsdk:"layout"`
	Template *observabilityDashboardTemplateModel `tfsdk:"template"`
	Group    *observabilityGroupModel             `tfsdk:"group"`
}

type observabilityGroupModel struct {
	Title      types.String                       `tfsdk:"title"`
	Headerless types.Bool                         `tfsdk:"headerless"`
	Layout     *observabilityLayoutOptionsModel   `tfsdk:"layout"`
	Container  []observabilityGroupContainerModel `tfsdk:"container"`
}

type observabilityGroupContainerModel struct {
	Layout   *observabilityLayoutModel            `tfsdk:"layout"`
	Template *observabilityDashboardTemplateModel `tfsdk:"template"`
}

// observabilityContainer is the shared semantic model used after decoding and
// before encoding the level-specific Terraform transport models.
type observabilityContainer struct {
	Layout   *observabilityLayoutModel
	Template *observabilityDashboardTemplateModel
	Section  *observabilitySection
	Group    *observabilityGroup
	// TODO(charts): Carry the generated chart-content abstraction through this
	// shared model so metrics_single_value, metrics_timeseries, and later schema
	// additions do not require hand-written fields here.
}

type observabilitySection struct {
	Title       types.String
	Collapse    types.Bool
	Collapsible types.Bool
	Layout      *observabilityLayoutOptionsModel
	Container   []observabilityContainer
}

type observabilityGroup struct {
	Title      types.String
	Headerless types.Bool
	Layout     *observabilityLayoutOptionsModel
	Container  []observabilityContainer
}

type observabilityContainerLevel uint8

const (
	observabilityDashboardContainerLevel observabilityContainerLevel = iota
	observabilitySectionContainerLevel
	observabilityGroupContainerLevel
)

func observabilityContainersFromDashboardModels(models []observabilityDashboardContainerModel) []observabilityContainer {
	containers := make([]observabilityContainer, len(models))
	for i, model := range models {
		container := observabilityContainer{Layout: model.Layout, Template: model.Template}
		if model.Section != nil {
			container.Section = &observabilitySection{
				Title:       model.Section.Title,
				Collapse:    model.Section.Collapse,
				Collapsible: model.Section.Collapsible,
				Layout:      model.Section.Layout,
				Container:   observabilityContainersFromSectionModels(model.Section.Container),
			}
		}
		if model.Group != nil {
			container.Group = observabilityGroupFromModel(model.Group)
		}
		containers[i] = container
	}
	return containers
}

func observabilityContainersFromSectionModels(models []observabilitySectionContainerModel) []observabilityContainer {
	containers := make([]observabilityContainer, len(models))
	for i, model := range models {
		container := observabilityContainer{Layout: model.Layout, Template: model.Template}
		if model.Group != nil {
			container.Group = observabilityGroupFromModel(model.Group)
		}
		containers[i] = container
	}
	return containers
}

func observabilityGroupFromModel(model *observabilityGroupModel) *observabilityGroup {
	return &observabilityGroup{
		Title:      model.Title,
		Headerless: model.Headerless,
		Layout:     model.Layout,
		Container:  observabilityContainersFromGroupModels(model.Container),
	}
}

func observabilityContainersFromGroupModels(models []observabilityGroupContainerModel) []observabilityContainer {
	containers := make([]observabilityContainer, len(models))
	for i, model := range models {
		containers[i] = observabilityContainer{Layout: model.Layout, Template: model.Template}
	}
	return containers
}

func observabilityDashboardModelsFromContainers(containers []observabilityContainer) []observabilityDashboardContainerModel {
	models := make([]observabilityDashboardContainerModel, len(containers))
	for i, container := range containers {
		model := observabilityDashboardContainerModel{Layout: container.Layout, Template: container.Template}
		if container.Section != nil {
			model.Section = &observabilitySectionModel{
				Title:       container.Section.Title,
				Collapse:    container.Section.Collapse,
				Collapsible: container.Section.Collapsible,
				Layout:      container.Section.Layout,
				Container:   observabilitySectionModelsFromContainers(container.Section.Container),
			}
		}
		if container.Group != nil {
			model.Group = observabilityGroupModelFromGroup(container.Group)
		}
		models[i] = model
	}
	return models
}

func observabilitySectionModelsFromContainers(containers []observabilityContainer) []observabilitySectionContainerModel {
	models := make([]observabilitySectionContainerModel, len(containers))
	for i, container := range containers {
		model := observabilitySectionContainerModel{Layout: container.Layout, Template: container.Template}
		if container.Group != nil {
			model.Group = observabilityGroupModelFromGroup(container.Group)
		}
		models[i] = model
	}
	return models
}

func observabilityGroupModelFromGroup(group *observabilityGroup) *observabilityGroupModel {
	return &observabilityGroupModel{
		Title:      group.Title,
		Headerless: group.Headerless,
		Layout:     group.Layout,
		Container:  observabilityGroupModelsFromContainers(group.Container),
	}
}

func observabilityGroupModelsFromContainers(containers []observabilityContainer) []observabilityGroupContainerModel {
	models := make([]observabilityGroupContainerModel, len(containers))
	for i, container := range containers {
		models[i] = observabilityGroupContainerModel{Layout: container.Layout, Template: container.Template}
	}
	return models
}
