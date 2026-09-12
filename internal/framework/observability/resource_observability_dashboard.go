// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityDashboardResource struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

var (
	_ resource.Resource              = (*observabilityDashboardResource)(nil)
	_ resource.ResourceWithConfigure = (*observabilityDashboardResource)(nil)
)

func NewResourceObservabilityDashboard() resource.Resource {
	return &observabilityDashboardResource{}
}

func (r *observabilityDashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_dashboard"
}

func (r *observabilityDashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceData.Configure(ctx, req, resp)
}

func (r *observabilityDashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability dashboard Template using reusable Template references. Typed inline charts and Directory placement are coming soon.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Dashboard title.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"control_bar": observabilityControlBarBlock(),
			"layout":      observabilityLayoutOptionsBlock(),
			"container": schema.ListNestedBlock{
				Description: "Ordered dashboard contents. Terraform declaration order is authoritative.",
				NestedObject: schema.NestedBlockObject{
					Blocks: observabilityContainerBlocks(observabilityDashboardContainerLevel),
				},
			},
		},
	}
}

func observabilityContainerBlocks(level observabilityContainerLevel) map[string]schema.Block {
	// TODO(charts): Merge the blocks emitted by the external-schema chart
	// generator here, including metrics_single_value and metrics_timeseries.
	blocks := map[string]schema.Block{
		"layout": observabilityItemLayoutBlock(),
		"template": schema.SingleNestedBlock{
			Description: "Reference to a reusable Observability Template rendered in this container.",
			Attributes: map[string]schema.Attribute{
				"template_id": schema.StringAttribute{Optional: true, Description: "ID of the referenced Template."},
			},
		},
	}
	if level == observabilityDashboardContainerLevel {
		blocks["section"] = schema.SingleNestedBlock{
			Description: "A titled section containing containers and optional groups.",
			Attributes: map[string]schema.Attribute{
				"title":       schema.StringAttribute{Optional: true, Description: "Section title."},
				"collapse":    schema.BoolAttribute{Optional: true, Description: "Whether the section is currently collapsed."},
				"collapsible": schema.BoolAttribute{Optional: true, Description: "Whether the section can be collapsed."},
			},
			Blocks: map[string]schema.Block{
				"layout": observabilityLayoutOptionsBlock(),
				"container": schema.ListNestedBlock{
					Description: "Ordered contents of this section.",
					NestedObject: schema.NestedBlockObject{
						Blocks: observabilityContainerBlocks(observabilitySectionContainerLevel),
					},
				},
			},
		}
	}
	if level != observabilityGroupContainerLevel {
		blocks["group"] = schema.SingleNestedBlock{
			Description: "A group containing related containers.",
			Attributes: map[string]schema.Attribute{
				"title":      schema.StringAttribute{Optional: true, Description: "Group title."},
				"headerless": schema.BoolAttribute{Optional: true, Description: "Whether to hide the group header."},
			},
			Blocks: map[string]schema.Block{
				"layout": observabilityLayoutOptionsBlock(),
				"container": schema.ListNestedBlock{
					Description: "Ordered contents of this group.",
					NestedObject: schema.NestedBlockObject{
						Blocks: observabilityContainerBlocks(observabilityGroupContainerLevel),
					},
				},
			},
		}
	}
	return blocks
}

func observabilityItemLayoutBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode.",
		Attributes: map[string]schema.Attribute{
			"absolute":   schema.BoolAttribute{Optional: true, Description: "Whether to position the container independently using its x and y coordinates."},
			"width":      observabilityLayoutLengthAttribute("Starting width of the container."),
			"height":     observabilityLayoutLengthAttribute("Starting height of the container."),
			"min_width":  observabilityLayoutLengthAttribute("Minimum width of the container."),
			"max_width":  observabilityLayoutLengthAttribute("Maximum width of the container."),
			"min_height": observabilityLayoutLengthAttribute("Minimum height of the container."),
			"max_height": observabilityLayoutLengthAttribute("Maximum height of the container."),
			"x":          observabilityLayoutLengthAttribute("Horizontal coordinate. A jsonencoded array is treated as a sum of lengths."),
			"y":          observabilityLayoutLengthAttribute("Vertical coordinate. A jsonencoded array is treated as a sum of lengths."),
		},
	}
}

func observabilityLayoutOptionsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Settings for the layout that arranges this level's containers.",
		Attributes: map[string]schema.Attribute{
			"gap": schema.Float64Attribute{
				Optional:    true,
				Description: "Visual gap between adjacent containers in pixels.",
				Validators:  []validator.Float64{float64validator.AtLeast(0)},
			},
			"step": schema.Float64Attribute{
				Optional:    true,
				Description: "Layout resolution in pixels. Lengths are rounded to multiples of this value.",
				Validators:  []validator.Float64{float64validator.AtLeast(1)},
			},
		},
		Blocks: map[string]schema.Block{
			"defaults": schema.SingleNestedBlock{
				Description: "Default placement and size constraints inherited by every container in this layout.",
				Attributes:  observabilityLayoutDefaultAttributes(),
			},
		},
	}
}

func observabilityLayoutDefaultAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"absolute":   schema.BoolAttribute{Optional: true, Description: "Default absolute-positioning behavior."},
		"width":      observabilityLayoutLengthAttribute("Default starting width."),
		"height":     observabilityLayoutLengthAttribute("Default starting height."),
		"min_width":  observabilityLayoutLengthAttribute("Default minimum width."),
		"max_width":  observabilityLayoutLengthAttribute("Default maximum width."),
		"min_height": observabilityLayoutLengthAttribute("Default minimum height."),
		"max_height": observabilityLayoutLengthAttribute("Default maximum height."),
	}
}

func observabilityLayoutLengthAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Description: description}
}

func (r *observabilityDashboardResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityDashboardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateObservabilityTitle(resp, path.Root("title"), model.Title)
	validateObservabilityControlBar(resp, path.Root("control_bar"), model.ControlBar)
	validateObservabilityLayoutOptions(resp, path.Root("layout"), model.Layout)
	validateObservabilityContainers(
		resp,
		path.Root("container"),
		observabilityContainersFromDashboardModels(model.Container),
		observabilityDashboardContainerLevel,
	)
}

func validateObservabilityContainers(resp *resource.ValidateConfigResponse, base path.Path, containers []observabilityContainer, level observabilityContainerLevel) {
	for i, container := range containers {
		containerPath := base.AtListIndex(i)
		validateObservabilityLayout(resp, containerPath.AtName("layout"), container.Layout)

		// TODO(charts): Include the generated ChartContents result in this one-of
		// count and validate each chart's generated required-field metadata.
		contentCount := 0
		if container.Template != nil {
			contentCount++
		}
		if container.Section != nil {
			contentCount++
		}
		if container.Group != nil {
			contentCount++
		}
		if contentCount != 1 || !observabilityContainerContentAllowed(container, level) {
			resp.Diagnostics.AddAttributeError(containerPath, "Invalid container content", observabilityContainerContentError(level))
			continue
		}

		if container.Template != nil {
			validateObservabilityTemplateReference(resp, containerPath, container.Template)
		}
		if container.Section != nil {
			sectionPath := containerPath.AtName("section")
			validateObservabilityLayoutOptions(resp, sectionPath.AtName("layout"), container.Section.Layout)
			validateObservabilityTitle(resp, sectionPath.AtName("title"), container.Section.Title)
			if len(container.Section.Container) == 0 {
				resp.Diagnostics.AddAttributeError(sectionPath, "Empty section", "section must contain at least one container")
			}
			validateObservabilityContainers(resp, sectionPath.AtName("container"), container.Section.Container, observabilitySectionContainerLevel)
		}
		if container.Group != nil {
			groupPath := containerPath.AtName("group")
			validateObservabilityLayoutOptions(resp, groupPath.AtName("layout"), container.Group.Layout)
			validateObservabilityTitle(resp, groupPath.AtName("title"), container.Group.Title)
			if len(container.Group.Container) == 0 {
				resp.Diagnostics.AddAttributeError(groupPath, "Empty group", "group must contain at least one container")
			}
			validateObservabilityContainers(resp, groupPath.AtName("container"), container.Group.Container, observabilityGroupContainerLevel)
		}
	}
}

func observabilityContainerContentAllowed(container observabilityContainer, level observabilityContainerLevel) bool {
	switch level {
	case observabilityDashboardContainerLevel:
		return container.Template != nil || container.Section != nil || container.Group != nil
	case observabilitySectionContainerLevel:
		return container.Template != nil || container.Group != nil
	case observabilityGroupContainerLevel:
		return container.Template != nil
	default:
		return false
	}
}

func observabilityContainerContentError(level observabilityContainerLevel) string {
	switch level {
	case observabilityDashboardContainerLevel:
		return "each dashboard container must set exactly one content block: template, section, or group"
	case observabilitySectionContainerLevel:
		return "each container within a section must set exactly one content block: template or group"
	case observabilityGroupContainerLevel:
		return "each container within a group must set exactly one template block"
	default:
		return "each container must set exactly one supported content block"
	}
}

func validateObservabilityLayout(resp *resource.ValidateConfigResponse, layoutPath path.Path, layout *observabilityLayoutModel) {
	if layout == nil {
		return
	}
	if observabilityLayoutIsEmpty(layout) {
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout block", "layout must set at least one placement or size option")
		return
	}
	for _, field := range []struct {
		name       string
		value      types.String
		coordinate bool
	}{
		{name: "width", value: layout.Width},
		{name: "height", value: layout.Height},
		{name: "min_width", value: layout.MinWidth},
		{name: "max_width", value: layout.MaxWidth},
		{name: "min_height", value: layout.MinHeight},
		{name: "max_height", value: layout.MaxHeight},
		{name: "x", value: layout.X, coordinate: true},
		{name: "y", value: layout.Y, coordinate: true},
	} {
		validateObservabilityLayoutValue(resp, layoutPath.AtName(field.name), field.value, field.coordinate)
	}
}

func validateObservabilityLayoutOptions(resp *resource.ValidateConfigResponse, layoutPath path.Path, layout *observabilityLayoutOptionsModel) {
	if layout == nil {
		return
	}
	if layout.Gap.IsNull() && layout.Step.IsNull() && layout.Defaults == nil {
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout block", "layout must set gap, step, defaults, or a combination")
		return
	}
	if layout.Defaults == nil {
		return
	}
	defaultsPath := layoutPath.AtName("defaults")
	if observabilityLayoutDefaultsAreEmpty(layout.Defaults) {
		resp.Diagnostics.AddAttributeError(defaultsPath, "Empty defaults block", "defaults must set at least one placement or size option")
		return
	}
	for _, field := range []struct {
		name  string
		value types.String
	}{
		{name: "width", value: layout.Defaults.Width},
		{name: "height", value: layout.Defaults.Height},
		{name: "min_width", value: layout.Defaults.MinWidth},
		{name: "max_width", value: layout.Defaults.MaxWidth},
		{name: "min_height", value: layout.Defaults.MinHeight},
		{name: "max_height", value: layout.Defaults.MaxHeight},
	} {
		validateObservabilityLayoutValue(resp, defaultsPath.AtName(field.name), field.value, false)
	}
}

func validateObservabilityLayoutValue(resp *resource.ValidateConfigResponse, valuePath path.Path, value types.String, coordinate bool) {
	if _, _, err := observabilityLayoutValue(value, coordinate); err != nil {
		resp.Diagnostics.AddAttributeError(valuePath, "Invalid layout value", err.Error())
	}
}

func observabilityLayoutIsEmpty(layout *observabilityLayoutModel) bool {
	return layout.Absolute.IsNull() &&
		layout.Width.IsNull() && layout.Height.IsNull() &&
		layout.MinWidth.IsNull() && layout.MaxWidth.IsNull() &&
		layout.MinHeight.IsNull() && layout.MaxHeight.IsNull() &&
		layout.X.IsNull() && layout.Y.IsNull()
}

func observabilityLayoutDefaultsAreEmpty(defaults *observabilityLayoutDefaultsModel) bool {
	return defaults.Absolute.IsNull() &&
		defaults.Width.IsNull() && defaults.Height.IsNull() &&
		defaults.MinWidth.IsNull() && defaults.MaxWidth.IsNull() &&
		defaults.MinHeight.IsNull() && defaults.MaxHeight.IsNull()
}

func validateObservabilityTemplateReference(resp *resource.ValidateConfigResponse, containerPath path.Path, ref *observabilityTemplateReferenceModel) {
	if !ref.TemplateID.IsUnknown() && (ref.TemplateID.IsNull() || ref.TemplateID.ValueString() == "") {
		resp.Diagnostics.AddAttributeError(containerPath.AtName("template").AtName("template_id"), "Missing required value", "template_id must be set when template is used")
	}
}

func (r *observabilityDashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model observabilityDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	write, err := observabilityDashboardTemplateWrite(model)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dashboard", err.Error())
		return
	}
	result, err := r.Details().Client.CreateTemplate(ctx, write)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dashboard", err.Error())
		return
	}
	model.ID = types.StringValue(record.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityDashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.GetTemplate(ctx, state.ID.ValueString(), nil)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	model, diags := parseDashboardTemplate(record)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model observabilityDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = state.ID
	write, err := observabilityDashboardTemplateWrite(model)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dashboard", err.Error())
		return
	}
	result, err := r.Details().Client.UpdateTemplate(ctx, model.ID.ValueString(), write)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dashboard", err.Error())
		return
	}
	model.ID = types.StringValue(record.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteTemplate(ctx, state.ID.ValueString()))...)
}

func observabilityDashboardTemplateWrite(model observabilityDashboardModel) (*template.Write, error) {
	spec, imports, err := buildDashboardSpec(model)
	if err != nil {
		return nil, err
	}
	rootElement := template.RootElementDashboard
	return &template.Write{
		Type:  template.RecordType,
		Title: model.Title.ValueString(),
		Spec:  spec,
		Metadata: template.WriteMetadata{
			RootElement: &rootElement,
			Imports:     imports,
		},
	}, nil
}

func unsupportedDashboardSpec(message string) diag.Diagnostics {
	return diag.Diagnostics{diag.NewErrorDiagnostic("Unsupported dashboard template", message)}
}

func formatDashboardParseError(err error) diag.Diagnostics {
	return unsupportedDashboardSpec(fmt.Sprintf("%v", err))
}
