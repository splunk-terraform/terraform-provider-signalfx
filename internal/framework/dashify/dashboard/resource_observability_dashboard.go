// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
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
		Description: "Manages an Observability dashboard Template using reusable Template references or raw inline dashboard content. Updates replace the saved dashboard content and discard unsupported fields. Typed chart attributes are coming soon; Directory placement is managed by signalfx_observability_directory.",
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
			"container":   dashifyContainerListAttribute(),
		},
	}
}

func dashifyContainerListAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "Ordered dashboard panels. Terraform declaration order is authoritative.",
		Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"template_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of a reusable Observability Template rendered in this panel.",
			},
			"inline_content": schema.StringAttribute{
				Optional:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "Self-contained dashboard JSON object rendered inline in this panel.",
			},
			"layout": dashifyItemLayoutAttribute(),
		}},
	}
}

func dashifyItemLayoutAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Panel sizing and position. Numeric values are layout units, not pixels.",
		Attributes: map[string]schema.Attribute{
			"absolute":   schema.BoolAttribute{Optional: true, Description: "Position the panel outside normal row flow using x and y. Defaults to normal flow when omitted. Absolute panels can overlap."},
			"x":          schema.StringAttribute{Optional: true, Description: "Starting horizontal position of the panel. Use 0 for the left edge, or a fraction or percentage such as 1/2 or 50%."},
			"y":          schema.StringAttribute{Optional: true, Description: "Starting vertical position in layout units. Takes effect when absolute is true; normal row flow calculates vertical placement automatically. Fractions and percentages are relative to canvas width, not height."},
			"width":      schema.StringAttribute{Optional: true, Description: "Starting width of the panel."},
			"height":     schema.StringAttribute{Optional: true, Description: "Starting height of the panel."},
			"min_width":  schema.StringAttribute{Optional: true, Description: "Minimum width of the panel."},
			"min_height": schema.StringAttribute{Optional: true, Description: "Minimum height of the panel."},
		},
	}
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
	fwdashify.ValidateObservabilityTitle(&resp, path.Root("title"), model.Title)
	validateDashifyControlBar(&resp, path.Root("control_bar"), model.ControlBar)
	for i := range model.Container {
		containerPath := path.Root("container").AtListIndex(i)
		validateDashifyLayout(&resp, containerPath.AtName("layout"), model.Container[i].Layout)
		validateDashifyTemplate(&resp, containerPath, &model.Container[i])
	}
	return resp.Diagnostics
}

func validateDashifyLayout(resp *resource.ValidateConfigResponse, layoutPath path.Path, layout *dashifyLayoutModel) {
	if layout == nil {
		return
	}
	if layout.Absolute.IsNull() && dashifyLayoutFieldsEmpty(dashifyLayoutModelFields(layout)) {
		resp.Diagnostics.AddAttributeError(layoutPath, "Empty layout", "layout must set at least one size or position option")
	}
}

func validateDashifyTemplate(resp *resource.ValidateConfigResponse, containerPath path.Path, model *dashifyDashboardContainerModel) {
	idKnown := !model.TemplateID.IsUnknown()
	contentKnown := !model.InlineContent.IsUnknown()
	idSet := idKnown && !model.TemplateID.IsNull()
	contentSet := contentKnown && !model.InlineContent.IsNull()

	if idKnown && contentKnown && idSet == contentSet {
		resp.Diagnostics.AddAttributeError(containerPath, "Invalid panel content", "container must set exactly one of template_id or inline_content")
		return
	}
	if idSet {
		id := model.TemplateID.ValueString()
		if id == "" {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Missing required value", "template_id must be non-empty when set")
		} else if !fwdashify.ObservabilityTemplateIDValid(id) {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Invalid Template ID", "Use a Template ID without an API path.")
		}
	}
	if contentSet {
		if _, err := decodeDashifyInlineContent(model.InlineContent.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("inline_content"), "Invalid inline content", err.Error())
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
	record, err := fwdashify.ObservabilityTemplateFromResult(result)
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

func observabilityDashboardTemplateWrite(model observabilityDashboardModel) (*template.CreateUpdateTemplateRequest, error) {
	spec, imports, err := buildDashboardSpec(model)
	if err != nil {
		return nil, err
	}
	rootElement := template.RootElementDashboard
	return &template.CreateUpdateTemplateRequest{
		Type:  fwdashify.ObservabilityTemplateRecordType,
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
