// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
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

func (r *observabilityDashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability dashboard Template using reusable Template references or raw inline dashboard content. Typed chart blocks are coming soon; Directory placement is managed by signalfx_observability_directory.",
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
			"control_bar": dashifyControlBarBlock(),
			"layout":      dashifyLayoutOptionsBlock(),
			"container": schema.ListNestedBlock{
				Description: "Ordered dashboard contents. Terraform declaration order is authoritative.",
				NestedObject: schema.NestedBlockObject{
					Attributes: dashifyContainerAttributes(),
					Blocks:     dashifyContainerBlocks(dashifyDashboardContainerLevel),
				},
			},
		},
	}
}

// dashifyContainerLevelRule is the single source of truth for which
// content blocks a container may hold at each nesting level: it drives
// schema block registration, the ValidateConfig one-of check, and the
// resulting error message, so the three stay in sync by construction
// instead of via three hand-maintained switches.
type dashifyContainerLevelRule struct {
	allowSection bool
	allowGroup   bool
	errorMessage string
	// Describes the stored elements accepted at this level, for Read errors.
	supportedElements string
}

var dashifyContainerLevelRules = map[dashifyContainerLevel]dashifyContainerLevelRule{
	dashifyDashboardContainerLevel: {
		allowSection:      true,
		allowGroup:        true,
		errorMessage:      "each dashboard container must set exactly one content source: template_id, template_content, section, or group",
		supportedElements: "panels, sections, and groups",
	},
	dashifySectionContainerLevel: {
		allowGroup:        true,
		errorMessage:      "each container within a section must set exactly one content source: template_id, template_content, or group",
		supportedElements: "panels and groups",
	},
	dashifyGroupContainerLevel: {
		errorMessage:      "each container within a group must set exactly one of template_id or template_content",
		supportedElements: "panels",
	},
}

func dashifyContainerAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"template_id": schema.StringAttribute{
			Optional:    true,
			Description: "ID of a reusable Observability Template rendered in this container.",
		},
		"template_content": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: "Self-contained dashboard JSON object rendered inline in this container.",
		},
	}
}

func dashifyContainerBlocks(level dashifyContainerLevel) map[string]schema.Block {
	// TODO(charts): Merge the blocks emitted by the external-schema chart
	// generator here, including metrics_single_value and metrics_timeseries.
	blocks := map[string]schema.Block{"layout": dashifyItemLayoutBlock()}
	rule := dashifyContainerLevelRules[level]
	if rule.allowSection {
		blocks["section"] = schema.SingleNestedBlock{
			Description: "A section containing containers and optional groups.",
			Attributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
					Description: "Optional section title. Omit it for an untitled section.",
				},
				"collapse":    schema.BoolAttribute{Optional: true, Description: "Whether the section is currently collapsed."},
				"collapsible": schema.BoolAttribute{Optional: true, Description: "Whether the section can be collapsed."},
			},
			Blocks: map[string]schema.Block{
				"layout": dashifyLayoutOptionsBlock(),
				"container": schema.ListNestedBlock{
					Description: "Ordered contents of this section.",
					NestedObject: schema.NestedBlockObject{
						Attributes: dashifyContainerAttributes(),
						Blocks:     dashifyContainerBlocks(dashifySectionContainerLevel),
					},
				},
			},
		}
	}
	if rule.allowGroup {
		blocks["group"] = schema.SingleNestedBlock{
			Description: "A group containing related containers.",
			Attributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
					Description: "Optional group title. Omit it for an untitled group.",
				},
				"headerless": schema.BoolAttribute{Optional: true, Description: "Whether to hide the group header."},
			},
			Blocks: map[string]schema.Block{
				"layout": dashifyLayoutOptionsBlock(),
				"container": schema.ListNestedBlock{
					Description: "Ordered contents of this group.",
					NestedObject: schema.NestedBlockObject{
						Attributes: dashifyContainerAttributes(),
						Blocks:     dashifyContainerBlocks(dashifyGroupContainerLevel),
					},
				},
			},
		}
	}
	return blocks
}

func dashifyItemLayoutBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Placement and size of this container inside its parent layout. Lengths accept numbers or relative strings; clamped values and coordinate arrays can be supplied with jsonencode.",
		Attributes: map[string]schema.Attribute{
			"order":      schema.Float64Attribute{Optional: true, Description: "Display order of the container within its parent layout, overriding declaration order. Reordering in the UI sets this on every container in the layout."},
			"absolute":   schema.BoolAttribute{Optional: true, Description: "Whether to position the container independently using its x and y coordinates."},
			"width":      dashifyLayoutLengthAttribute("Starting width of the container."),
			"height":     dashifyLayoutLengthAttribute("Starting height of the container."),
			"min_width":  dashifyLayoutLengthAttribute("Minimum width of the container."),
			"max_width":  dashifyLayoutLengthAttribute("Maximum width of the container."),
			"min_height": dashifyLayoutLengthAttribute("Minimum height of the container."),
			"max_height": dashifyLayoutLengthAttribute("Maximum height of the container."),
			"x":          dashifyLayoutLengthAttribute("Horizontal coordinate. A jsonencoded array is treated as a sum of lengths."),
			"y":          dashifyLayoutLengthAttribute("Vertical coordinate. A jsonencoded array is treated as a sum of lengths."),
		},
	}
}

func dashifyLayoutOptionsBlock() schema.SingleNestedBlock {
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
				Attributes:  dashifyLayoutDefaultAttributes(),
			},
		},
	}
}

func dashifyLayoutDefaultAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"absolute":   schema.BoolAttribute{Optional: true, Description: "Default absolute-positioning behavior."},
		"width":      dashifyLayoutLengthAttribute("Default starting width."),
		"height":     dashifyLayoutLengthAttribute("Default starting height."),
		"min_width":  dashifyLayoutLengthAttribute("Default minimum width."),
		"max_width":  dashifyLayoutLengthAttribute("Default maximum width."),
		"min_height": dashifyLayoutLengthAttribute("Default minimum height."),
		"max_height": dashifyLayoutLengthAttribute("Default maximum height."),
	}
}

func dashifyLayoutLengthAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Description: description}
}

func (r *observabilityDashboardResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityDashboardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateObservabilityTitle(resp, path.Root("title"), model.Title)
	validateDashifyControlBar(resp, path.Root("control_bar"), model.ControlBar)
	validateDashifyLayoutOptions(resp, path.Root("layout"), model.Layout)
	validateDashifyContainers(
		resp,
		path.Root("container"),
		dashifyContainersFromDashboardModels(model.Container),
		dashifyDashboardContainerLevel,
	)
}

func validateDashifyContainers(resp *resource.ValidateConfigResponse, base path.Path, containers []dashifyContainer, level dashifyContainerLevel) {
	for i, container := range containers {
		containerPath := base.AtListIndex(i)
		validateDashifyLayout(resp, containerPath.AtName("layout"), container.Layout)

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
		if contentCount != 1 || !dashifyContainerContentAllowed(container, level) {
			resp.Diagnostics.AddAttributeError(containerPath, "Invalid container content", dashifyContainerContentError(level))
			continue
		}

		if container.Template != nil {
			validateDashifyTemplate(resp, containerPath, container.Template)
		}
		if container.Section != nil {
			sectionPath := containerPath.AtName("section")
			validateDashifyLayoutOptions(resp, sectionPath.AtName("layout"), container.Section.Layout)
			if len(container.Section.Container) == 0 {
				resp.Diagnostics.AddAttributeError(sectionPath, "Empty section", "section must contain at least one container")
			}
			validateDashifyContainers(resp, sectionPath.AtName("container"), container.Section.Container, dashifySectionContainerLevel)
		}
		if container.Group != nil {
			groupPath := containerPath.AtName("group")
			validateDashifyLayoutOptions(resp, groupPath.AtName("layout"), container.Group.Layout)
			if len(container.Group.Container) == 0 {
				resp.Diagnostics.AddAttributeError(groupPath, "Empty group", "group must contain at least one container")
			}
			validateDashifyContainers(resp, groupPath.AtName("container"), container.Group.Container, dashifyGroupContainerLevel)
		}
	}
}

func dashifyContainerContentAllowed(container dashifyContainer, level dashifyContainerLevel) bool {
	rule := dashifyContainerLevelRules[level]
	return container.Template != nil ||
		(container.Section != nil && rule.allowSection) ||
		(container.Group != nil && rule.allowGroup)
}

func dashifyContainerContentError(level dashifyContainerLevel) string {
	return dashifyContainerLevelRules[level].errorMessage
}

func validateDashifyLayout(resp *resource.ValidateConfigResponse, layoutPath path.Path, layout *dashifyLayoutModel) {
	if layout == nil {
		return
	}
	if dashifyLayoutIsEmpty(layout) {
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout block", "layout must set at least one placement or size option")
		return
	}
	for _, field := range dashifyLayoutModelFields(layout) {
		validateDashifyLayoutValue(resp, layoutPath.AtName(field.name), *field.value, field.coordinate)
	}
}

func validateDashifyLayoutOptions(resp *resource.ValidateConfigResponse, layoutPath path.Path, layout *dashifyLayoutOptionsModel) {
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
	if dashifyLayoutDefaultsAreEmpty(layout.Defaults) {
		resp.Diagnostics.AddAttributeError(defaultsPath, "Empty defaults block", "defaults must set at least one placement or size option")
		return
	}
	for _, field := range dashifyLayoutDefaultsModelFields(layout.Defaults) {
		validateDashifyLayoutValue(resp, defaultsPath.AtName(field.name), *field.value, field.coordinate)
	}
}

func validateDashifyLayoutValue(resp *resource.ValidateConfigResponse, valuePath path.Path, value types.String, coordinate bool) {
	if _, _, err := dashifyLayoutValue(value, coordinate); err != nil {
		resp.Diagnostics.AddAttributeError(valuePath, "Invalid layout value", err.Error())
	}
}

func dashifyLayoutIsEmpty(layout *dashifyLayoutModel) bool {
	return layout.Order.IsNull() && dashifyLayoutFieldsEmpty(layout.Absolute, dashifyLayoutModelFields(layout))
}

func dashifyLayoutDefaultsAreEmpty(defaults *dashifyLayoutDefaultsModel) bool {
	return dashifyLayoutFieldsEmpty(defaults.Absolute, dashifyLayoutDefaultsModelFields(defaults))
}

func validateDashifyTemplate(resp *resource.ValidateConfigResponse, containerPath path.Path, model *dashifyTemplateModel) {
	idKnown := !model.TemplateID.IsUnknown()
	contentKnown := !model.Content.IsUnknown()
	idSet := idKnown && !model.TemplateID.IsNull()
	contentSet := contentKnown && !model.Content.IsNull()

	if idKnown && contentKnown && idSet == contentSet {
		resp.Diagnostics.AddAttributeError(containerPath, "Invalid template content", "container must set exactly one of template_id or template_content")
		return
	}
	if idSet {
		id := model.TemplateID.ValueString()
		if id == "" {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Missing required value", "template_id must be non-empty when set")
		} else if !observabilityTemplateIDValid(id) {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Invalid Template ID", "Use a Template ID without an API path.")
		}
	}
	if contentSet {
		if _, err := decodeDashifyInlineContent(model.Content.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_content"), "Invalid inline content", err.Error())
		}
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
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.Diagnostics.AddError("Error creating dashboard", "The Template API endpoint was not found.")
		return
	}
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
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	if record.ID != state.ID.ValueString() {
		resp.Diagnostics.AddError("Error reading dashboard", fmt.Sprintf("Template API returned record %q when reading Dashboard %q.", record.ID, state.ID.ValueString()))
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
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.Diagnostics.AddError("Error updating dashboard", fmt.Sprintf("Dashboard %q was not found during update; retry after refreshing the Terraform state.", model.ID.ValueString()))
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dashboard", err.Error())
		return
	}
	if record.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Error updating dashboard", "The Template API returned a different dashboard ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	result, lookupErr := r.Details().Client.GetTemplate(ctx, id, nil)
	if lookupErr == nil && result != nil && result.Data != nil && len(result.Data.DirectoryEntries) > 0 {
		resp.Diagnostics.AddWarning(
			"Deleting dashboard with Directory memberships",
			fmt.Sprintf("The dashboard is referenced by %d Directory entry(s). Deleting it can leave dangling memberships in those Directories; update them after deletion.", len(result.Data.DirectoryEntries)),
		)
	}
	err := r.Details().Client.DeleteTemplate(ctx, id)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func observabilityDashboardTemplateWrite(model observabilityDashboardModel) (*template.CreateUpdateTemplateRequest, error) {
	spec, imports, err := buildDashboardSpec(model)
	if err != nil {
		return nil, err
	}
	rootElement := template.RootElementDashboard
	return &template.CreateUpdateTemplateRequest{
		Type:  observabilityTemplateRecordType,
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
