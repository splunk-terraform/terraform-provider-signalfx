// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityTemplateModel struct {
	ID          types.String                        `tfsdk:"id"`
	Title       types.String                        `tfsdk:"title"`
	RootElement types.String                        `tfsdk:"root_element"`
	Spec        types.String                        `tfsdk:"spec"`
	Metadata    *observabilityTemplateMetadataModel `tfsdk:"metadata"`
}

type observabilityTemplateMetadataModel struct {
	Imports    types.List                            `tfsdk:"imports"`
	Datasource *observabilityTemplateDatasourceModel `tfsdk:"datasource"`
}

type observabilityTemplateDatasourceModel struct {
	Type        types.String `tfsdk:"type"`
	ProgramText types.String `tfsdk:"program_text"`
	SLOID       types.String `tfsdk:"slo_id"`
}

type observabilityTemplateResource struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

var (
	_ resource.Resource              = (*observabilityTemplateResource)(nil)
	_ resource.ResourceWithConfigure = (*observabilityTemplateResource)(nil)
)

func NewResourceObservabilityTemplate() resource.Resource {
	return &observabilityTemplateResource{}
}

func (r *observabilityTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_template"
}

func (r *observabilityTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceData.Configure(ctx, req, resp)
}

func (r *observabilityTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability Template record using the Template API write model.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Template title.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"root_element": schema.StringAttribute{
				Required:    true,
				Description: "Non-empty root Dashify element name represented by the template, such as Chart or Dashboard. Changing this value replaces the resource because the API does not permit root element updates.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"spec": schema.StringAttribute{
				Required:    true,
				Description: "JSON object containing the polymorphic Dashify template specification.",
				PlanModifiers: []planmodifier.String{
					observabilityJSONSemanticEqualityModifier{},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Description: "Optional metadata extracted from the template specification.",
				Attributes: map[string]schema.Attribute{
					"imports": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Template record paths imported directly by the specification.",
					},
				},
				Blocks: map[string]schema.Block{
					"datasource": schema.SingleNestedBlock{
						Description: "Datasource metadata extracted from the template.",
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Optional:    true,
								Description: "Datasource type.",
								Validators: []validator.String{
									stringvalidator.OneOf(string(template.DatasourceTypeSplunkObservability), string(template.DatasourceTypeSplunkObservabilitySLO)),
								},
							},
							"program_text": schema.StringAttribute{
								Optional:    true,
								Description: "SignalFlow program text associated with the template.",
							},
							"slo_id": schema.StringAttribute{
								Optional:    true,
								Description: "Service-level objective ID. Required when type is SPLUNK_O11Y_SLO.",
							},
						},
					},
				},
			},
		},
	}
}

func (r *observabilityTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateObservabilityTitle(resp, path.Root("title"), model.Title)
	if !model.Spec.IsNull() && !model.Spec.IsUnknown() {
		if err := validateObservabilityTemplateSpec(model.Spec.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("spec"), "Invalid template specification", err.Error())
		}
	}
	if model.Metadata == nil || model.Metadata.Datasource == nil {
		return
	}
	validateObservabilityTemplateDatasource(model.RootElement, *model.Metadata.Datasource, resp)
}

func validateObservabilityTemplateDatasource(rootElement types.String, datasource observabilityTemplateDatasourceModel, resp *resource.ValidateConfigResponse) {
	if !datasource.Type.IsUnknown() && !datasource.SLOID.IsUnknown() &&
		datasource.Type.ValueString() == string(template.DatasourceTypeSplunkObservabilitySLO) &&
		(datasource.SLOID.IsNull() || datasource.SLOID.ValueString() == "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("metadata").AtName("datasource").AtName("slo_id"),
			"Missing SLO ID",
			"slo_id must be provided when datasource type is SPLUNK_O11Y_SLO.",
		)
	}
	if rootElement.IsNull() || rootElement.IsUnknown() ||
		(datasource.Type.IsUnknown() || datasource.ProgramText.IsUnknown()) {
		return
	}
	root := rootElement.ValueString()
	if !strings.EqualFold(root, string(template.RootElementChart)) &&
		!strings.EqualFold(root, string(template.RootElementDashboard)) {
		return
	}

	typePresent := !datasource.Type.IsNull()
	programPresent := !datasource.ProgramText.IsNull() && datasource.ProgramText.ValueString() != ""
	if programPresent && !typePresent {
		resp.Diagnostics.AddAttributeError(
			path.Root("metadata").AtName("datasource").AtName("type"),
			"Missing datasource type",
			"type must be provided when program_text is set on a Chart or Dashboard template.",
		)
	}
	if typePresent && !programPresent {
		resp.Diagnostics.AddAttributeError(
			path.Root("metadata").AtName("datasource").AtName("program_text"),
			"Missing SignalFlow program",
			"program_text must be non-empty when datasource type is set on a Chart or Dashboard template.",
		)
	}
}

func (r *observabilityTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	write, diags := observabilityTemplateWrite(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.CreateTemplate(ctx, write)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating template", err.Error())
		return
	}
	model.ID = types.StringValue(record.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.Details().Client.GetTemplate(ctx, model.ID.ValueString(), nil)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading template", err.Error())
		return
	}
	model, err = observabilityTemplateModelFromRecord(model, record)
	if err != nil {
		resp.Diagnostics.AddError("Error reading template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior observabilityTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = prior.ID

	write, diags := observabilityTemplateWrite(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.UpdateTemplate(ctx, model.ID.ValueString(), write)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if _, err := observabilityTemplateFromResult(result); err != nil {
		resp.Diagnostics.AddError("Error updating template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *observabilityTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteTemplate(ctx, model.ID.ValueString()))...)
}

type observabilityJSONSemanticEqualityModifier struct{}

func (observabilityJSONSemanticEqualityModifier) Description(_ context.Context) string {
	return "Treats JSON content as unchanged when it is semantically equivalent to the prior value."
}

func (m observabilityJSONSemanticEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m observabilityJSONSemanticEqualityModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if observabilityJSONEqual(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
