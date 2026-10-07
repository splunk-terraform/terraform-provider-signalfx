// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

const (
	observabilityDashboardCompleteWriteKnownKey = "complete_write_known"
	observabilityDashboardUnsafeUpdateSummary   = "Cannot safely update imported or unmodeled dashboard"
	observabilityDashboardUnsafeUpdateDetail    = "The dashboard may contain fields or write metadata this resource cannot recover. Updating it would replace the complete document and could remove them. To manage this dashboard, import it as signalfx_observability_template with its complete specification and explicit metadata instead."
)

var (
	_ resource.Resource               = (*observabilityDashboardResource)(nil)
	_ resource.ResourceWithConfigure  = (*observabilityDashboardResource)(nil)
	_ resource.ResourceWithModifyPlan = (*observabilityDashboardResource)(nil)
)

func NewResourceObservabilityDashboard() resource.Resource {
	return &observabilityDashboardResource{}
}

func (r *observabilityDashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_dashboard"
}

func (r *observabilityDashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability dashboard Template using reusable Template references or raw inline dashboard content. Typed chart attributes are coming soon; Directory placement is managed by signalfx_observability_directory.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Dashboard title.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"control_bar": dashifyControlBarAttribute(),
			"layout":      dashifyLayoutOptionsAttribute(),
			"container":   dashifyContainerListAttribute("Ordered dashboard contents. Terraform declaration order is authoritative.", dashifyDashboardContainerLevel, false),
		},
	}
}

// dashifyContainerLevelRule is the single source of truth for which
// content a container may hold at each nesting level: it drives schema
// attribute registration, the ValidateConfig one-of check, and the
// resulting error message, so the three stay in sync by construction.
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

func dashifyContainerListAttribute(description string, level dashifyContainerLevel, required bool) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:     required,
		Optional:     !required,
		Description:  description,
		Validators:   []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedAttributeObject{Attributes: dashifyContainerAttributes(level)},
	}
}

func dashifyContainerAttributes(level dashifyContainerLevel) map[string]schema.Attribute {
	// TODO(charts): Merge the attributes emitted by the external-schema chart
	// generator here, including metrics_single_value and metrics_timeseries.
	attributes := map[string]schema.Attribute{
		"template_id": schema.StringAttribute{
			Optional:    true,
			Description: "ID of a reusable Observability Template rendered in this container.",
		},
		"template_content": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: "Self-contained dashboard JSON object rendered inline in this container.",
		},
		"layout": dashifyItemLayoutAttribute(),
	}
	rule := dashifyContainerLevelRules[level]
	if rule.allowSection {
		attributes["section"] = schema.SingleNestedAttribute{
			Optional:    true,
			Description: "A section containing containers and optional groups.",
			Attributes: map[string]schema.Attribute{
				"title":       dashifyOptionalTitleAttribute("Optional section title. Omit it for an untitled section."),
				"collapse":    schema.BoolAttribute{Optional: true, Description: "Whether the section is currently collapsed."},
				"collapsible": schema.BoolAttribute{Optional: true, Description: "Whether the section can be collapsed."},
				"layout":      dashifyLayoutOptionsAttribute(),
				"container":   dashifyContainerListAttribute("Ordered contents of this section.", dashifySectionContainerLevel, true),
			},
		}
	}
	if rule.allowGroup {
		attributes["group"] = schema.SingleNestedAttribute{
			Optional:    true,
			Description: "A group containing related containers.",
			Attributes: map[string]schema.Attribute{
				"title":      dashifyOptionalTitleAttribute("Optional group title. Omit it for an untitled group."),
				"headerless": schema.BoolAttribute{Optional: true, Description: "Whether to hide the group header."},
				"layout":     dashifyLayoutOptionsAttribute(),
				"container":  dashifyContainerListAttribute("Ordered contents of this group.", dashifyGroupContainerLevel, true),
			},
		}
	}
	return attributes
}

func dashifyOptionalTitleAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(""),
		Description: description,
	}
}

func dashifyItemLayoutAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
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

func dashifyLayoutOptionsAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
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
			"defaults": schema.SingleNestedAttribute{
				Optional:    true,
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

// Validates each top-level value once it is fully known; Create and Update
// validate the complete model again after unknown values resolve.
func (r *observabilityDashboardResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	model := observabilityDashboardModel{Title: types.StringUnknown()}
	for _, field := range []struct {
		name   string
		target any
	}{
		{"title", &model.Title},
		{"control_bar", &model.ControlBar},
		{"layout", &model.Layout},
		{"container", &model.Container},
	} {
		value, _, err := tftypes.WalkAttributePath(req.Config.Raw, tftypes.NewAttributePath().WithAttributeName(field.name))
		if known, ok := value.(tftypes.Value); err != nil || !ok || !known.IsFullyKnown() {
			continue
		}
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(field.name), field.target)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateDashboardModel(model)...)
}

func validateDashboardModel(model observabilityDashboardModel) diag.Diagnostics {
	var resp resource.ValidateConfigResponse
	validateObservabilityTitle(&resp, path.Root("title"), model.Title)
	validateDashifyControlBar(&resp, path.Root("control_bar"), model.ControlBar)
	validateDashifyLayoutOptions(&resp, path.Root("layout"), model.Layout)
	validateDashifyContainers(
		&resp,
		path.Root("container"),
		dashifyContainersFromDashboardModels(model.Container),
		dashifyDashboardContainerLevel,
	)
	return resp.Diagnostics
}

func (r *observabilityDashboardResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || req.Plan.Raw.Equal(req.State.Raw) {
		return
	}
	var knownWrite []byte
	if req.Private != nil {
		var diags diag.Diagnostics
		knownWrite, diags = req.Private.GetKey(ctx, observabilityDashboardCompleteWriteKnownKey)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() || len(knownWrite) > 0 {
		return
	}
	resp.Diagnostics.AddError(observabilityDashboardUnsafeUpdateSummary, observabilityDashboardUnsafeUpdateDetail)
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
			validateDashifyContainers(resp, sectionPath.AtName("container"), container.Section.Container, dashifySectionContainerLevel)
		}
		if container.Group != nil {
			groupPath := containerPath.AtName("group")
			validateDashifyLayoutOptions(resp, groupPath.AtName("layout"), container.Group.Layout)
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
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout", "layout must set at least one placement or size option")
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
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout", "layout must set gap, step, defaults, or a combination")
		return
	}
	if layout.Defaults == nil {
		return
	}
	defaultsPath := layoutPath.AtName("defaults")
	if dashifyLayoutDefaultsAreEmpty(layout.Defaults) {
		resp.Diagnostics.AddAttributeError(defaultsPath, "Empty defaults", "defaults must set at least one placement or size option")
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
	resp.Diagnostics.Append(validateDashboardModel(model)...)
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
	record, _, resultDiags := observabilityDashboardFromResult(result, "")
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = types.StringValue(record.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilityDashboardCompleteWriteKnownKey, dashboardKnownWriteMarker(resultDiags))...)
	}
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
	_, model, resultDiags := observabilityDashboardFromResult(result, state.ID.ValueString())
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if !resp.Diagnostics.HasError() && dashboardHasUnmodeledFields(resultDiags) {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilityDashboardCompleteWriteKnownKey, nil)...)
	}
}

func (r *observabilityDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model observabilityDashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateDashboardModel(model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var knownWrite []byte
	if req.Private != nil {
		var diags diag.Diagnostics
		knownWrite, diags = req.Private.GetKey(ctx, observabilityDashboardCompleteWriteKnownKey)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(knownWrite) == 0 {
		resp.Diagnostics.AddError(observabilityDashboardUnsafeUpdateSummary, observabilityDashboardUnsafeUpdateDetail)
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
	_, _, resultDiags := observabilityDashboardFromResult(result, model.ID.ValueString())
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilityDashboardCompleteWriteKnownKey, dashboardKnownWriteMarker(resultDiags))...)
	}
}

func (r *observabilityDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state observabilityDashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.Details().Client.DeleteTemplate(ctx, state.ID.ValueString())
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func observabilityDashboardFromResult(result *template.Result, expectedID string) (*template.Template, observabilityDashboardModel, diag.Diagnostics) {
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		return nil, observabilityDashboardModel{}, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Dashboard API response", err.Error())}
	}
	if expectedID != "" && record.ID != expectedID {
		return nil, observabilityDashboardModel{}, diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid Dashboard API response",
			fmt.Sprintf("Template API returned record %q for Dashboard %q.", record.ID, expectedID),
		)}
	}
	model, diags := parseDashboardTemplate(record)
	return record, model, diags
}

func dashboardHasUnmodeledFields(diags diag.Diagnostics) bool {
	for _, diagnostic := range diags {
		if diagnostic.Summary() == dashifyUnmodeledFieldsSummary {
			return true
		}
	}
	return false
}

func dashboardKnownWriteMarker(diags diag.Diagnostics) []byte {
	if dashboardHasUnmodeledFields(diags) {
		return nil
	}
	return []byte("true")
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
