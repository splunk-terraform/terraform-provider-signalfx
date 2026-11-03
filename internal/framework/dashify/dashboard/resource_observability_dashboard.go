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
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityDashboardResource struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
	engine      *interpreter.Engine
	schemaError error
}

var (
	_ resource.Resource                   = (*observabilityDashboardResource)(nil)
	_ resource.ResourceWithConfigure      = (*observabilityDashboardResource)(nil)
	_ resource.ResourceWithValidateConfig = (*observabilityDashboardResource)(nil)
)

func NewResourceObservabilityDashboard() resource.Resource {
	engine, err := interpreter.Default()
	return &observabilityDashboardResource{engine: engine, schemaError: err}
}

func (r *observabilityDashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_dashboard"
}

func (r *observabilityDashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.schemaError != nil {
		resp.Diagnostics.AddError("Invalid chart schema", r.schemaError.Error())
		return
	}
	containers, err := dashifyContainerListAttribute(r.engine)
	if err != nil {
		resp.Diagnostics.AddError("Invalid chart schema", err.Error())
		return
	}
	resp.Schema = schema.Schema{
		Description: "Manages an Observability dashboard with inline charts and shared Template references. Configure one typed chart, template_id, or raw inline_content per panel. Updates replace the saved dashboard document and discard unsupported dashboard fields. Directory placement is managed by signalfx_observability_directory.",
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
			"container":   containers,
		},
	}
}

func dashifyContainerListAttribute(engine *interpreter.Engine) (schema.ListNestedAttribute, error) {
	attributes, err := charts.Attributes(engine)
	if err != nil {
		return schema.ListNestedAttribute{}, err
	}
	attributes["inline_content"] = schema.StringAttribute{
		Optional: true, CustomType: jsontypes.NormalizedType{},
		Description: "Complete self-contained JSON object rendered inline in this panel. Use for content that the typed chart attributes cannot represent. External template imports are not allowed.",
	}
	attributes["layout"] = dashifyItemLayoutAttribute()
	attributes["template_id"] = schema.StringAttribute{
		Optional: true, Description: "ID of a reusable Observability Template rendered by reference in this panel. The dashboard does not manage the referenced Template's lifecycle.",
	}
	return schema.ListNestedAttribute{
		Optional:     true,
		Description:  "Ordered containers. Each container holds a typed inline chart, raw inline_content, or a template_id reference to a shared Template. Terraform declaration order is authoritative.",
		Validators:   []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedAttributeObject{Attributes: attributes},
	}, nil
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

// Validate known panel content even when another panel has computed values.
func (r *observabilityDashboardResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	model, diags := r.model(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(validateDashboardModel(r.engine, model)...)
	}
}

func validateDashboardModel(engine *interpreter.Engine, model observabilityDashboardModel) diag.Diagnostics {
	var resp resource.ValidateConfigResponse
	fwdashify.ValidateObservabilityTitle(&resp, path.Root("title"), model.Title)
	validateDashifyControlBar(&resp, path.Root("control_bar"), model.ControlBar)
	for i := range model.Container {
		containerPath := path.Root("container").AtListIndex(i)
		validateDashifyLayout(&resp, containerPath.AtName("layout"), model.Container[i].Layout)
		validateDashifyPanel(engine, &resp, containerPath, &model.Container[i])
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

func validateDashifyPanel(engine *interpreter.Engine, resp *resource.ValidateConfigResponse, containerPath path.Path, model *dashifyDashboardContainerModel) {
	contentKnown := !model.InlineContent.IsUnknown()
	contentSet := contentKnown && !model.InlineContent.IsNull()
	idKnown := !model.TemplateID.IsUnknown()
	idSet := idKnown && !model.TemplateID.IsNull()
	chartKnown := !model.ChartUnknown
	chartSet := model.Chart.Name != ""
	selected := 0
	for _, set := range []bool{contentSet, idSet, chartSet} {
		if set {
			selected++
		}
	}
	if selected > 1 || contentKnown && idKnown && chartKnown && selected != 1 {
		resp.Diagnostics.AddAttributeError(containerPath, "Invalid panel content", "Configure exactly one chart type, template_id, or inline_content per panel.")
		return
	}
	if idSet {
		if model.TemplateID.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Missing required value", "template_id must be non-empty when set")
		} else if !fwdashify.ObservabilityTemplateIDValid(model.TemplateID.ValueString()) {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("template_id"), "Invalid Template ID", "Use a Template ID without an API path.")
		}
	}
	if contentSet {
		if _, err := decodeDashifyInlineContent(model.InlineContent.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(containerPath.AtName("inline_content"), "Invalid inline content", err.Error())
		}
	}
	if model.Chart.Name != "" {
		for _, diagnostic := range engine.Validate(model.Chart.Name, model.Chart.Value) {
			resp.Diagnostics.AddAttributeError(containerPath.AtName(model.Chart.Name), diagnostic.Summary(), diagnostic.Detail())
		}
	}
}

func (r *observabilityDashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	model, diags := r.model(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateDashboardModel(r.engine, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	write, err := observabilityDashboardTemplateWrite(ctx, r.engine, model)
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
	record, resultDiags := dashboardRecordFromResult(result, "")
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = types.StringValue(record.ID)
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
}

func (r *observabilityDashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state, diags := r.model(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var options *template.GetOptions
	if len(state.Container) == 0 {
		// Imports can expand all panels. Ordinary refresh resolves only panels
		// that need inline content, without requesting shared chart records.
		options = &template.GetOptions{Imports: []template.Import{template.ImportAllChildren}}
	}
	result, err := r.Details().Client.GetTemplate(ctx, state.ID.ValueString(), options)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, resultDiags := dashboardRecordFromResult(result, state.ID.ValueString())
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolve := newDashboardChartResolver(ctx, r.Details().Client, result.Includes)
	model, resultDiags := parseDashboardTemplate(ctx, r.engine, record, state, resolve)
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = state.ID
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
}

func (r *observabilityDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	model, diags := r.model(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateDashboardModel(r.engine, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &model.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	write, err := observabilityDashboardTemplateWrite(ctx, r.engine, model)
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
	_, resultDiags := dashboardRecordFromResult(result, model.ID.ValueString())
	resp.Diagnostics.Append(resultDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
}

func (r *observabilityDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.Details().Client.DeleteTemplate(ctx, id.ValueString())
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func dashboardRecordFromResult(result *template.Result, expectedID string) (*template.Template, diag.Diagnostics) {
	record, err := fwdashify.ObservabilityTemplateFromResult(result)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Dashboard API response", err.Error())}
	}
	if !fwdashify.ObservabilityTemplateIDValid(record.ID) || expectedID != "" && record.ID != expectedID {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid Dashboard API response",
			fmt.Sprintf("Template API returned record %q for Dashboard %q.", record.ID, expectedID),
		)}
	}
	return record, nil
}

func observabilityDashboardTemplateWrite(ctx context.Context, engine *interpreter.Engine, model observabilityDashboardModel) (*template.CreateUpdateTemplateRequest, error) {
	spec, imports, err := buildDashboardSpec(ctx, engine, model)
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
