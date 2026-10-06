// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/template"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityTemplateModel struct {
	ID          types.String                        `tfsdk:"id"`
	Title       types.String                        `tfsdk:"title"`
	RootElement types.String                        `tfsdk:"root_element"`
	Spec        jsontypes.Normalized                `tfsdk:"spec"`
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

const (
	observabilityTemplateRecordType            = "#/dashify/v1/templates/Record"
	observabilityTemplateWriteMetadataKnownKey = "write_metadata_known"
	observabilityTemplateReferencePrefix       = signalfx.TemplateAPIURL + "/"
	observabilityTemplateUnsafeUpdateSummary   = "Cannot safely update imported template"
	observabilityTemplateUnsafeUpdateDetail    = "The Template API does not return all write-side metadata, so an update after import could erase existing imports or datasource metadata. Set metadata to the complete desired imports and datasource values before updating this template."
)

var (
	_ resource.Resource               = (*observabilityTemplateResource)(nil)
	_ resource.ResourceWithConfigure  = (*observabilityTemplateResource)(nil)
	_ resource.ResourceWithModifyPlan = (*observabilityTemplateResource)(nil)
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
				Description: "Non-empty root element name represented by the template, such as Chart or Dashboard. Changing this value replaces the resource because the API does not permit root element updates.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"spec": schema.StringAttribute{
				Required:    true,
				Description: "JSON object containing the polymorphic template specification.",
				CustomType:  jsontypes.NormalizedType{},
			},
			"metadata": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Optional metadata extracted from the template specification.",
				Attributes: map[string]schema.Attribute{
					"imports": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Template IDs imported directly by the specification.",
						Validators:  observabilityTemplateIDListValidators(),
					},
					"datasource": schema.SingleNestedAttribute{
						Optional:    true,
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

func (r *observabilityTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.Plan.Raw.Equal(req.State.Raw) || !req.Plan.Raw.IsFullyKnown() {
		return
	}

	// root_element is the only attribute that currently requires replacement.
	// A replacement creates a new Template and does not risk overwriting the
	// imported record's unmodeled write metadata.
	var priorRootElement, plannedRootElement types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("root_element"), &priorRootElement)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("root_element"), &plannedRootElement)...)
	if resp.Diagnostics.HasError() || priorRootElement.IsNull() || priorRootElement.IsUnknown() || plannedRootElement.IsNull() || plannedRootElement.IsUnknown() {
		return
	}
	if priorRootElement.ValueString() != plannedRootElement.ValueString() {
		return
	}

	if req.Private != nil {
		knownMetadata, diags := req.Private.GetKey(ctx, observabilityTemplateWriteMetadataKnownKey)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() || len(knownMetadata) > 0 {
			return
		}
	}

	var plannedMetadata types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &plannedMetadata)...)
	if resp.Diagnostics.HasError() || plannedMetadata.IsUnknown() || !plannedMetadata.IsNull() {
		return
	}

	resp.Diagnostics.AddError(observabilityTemplateUnsafeUpdateSummary, observabilityTemplateUnsafeUpdateDetail)
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
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.Diagnostics.AddError("Error creating template", "The Template API endpoint was not found.")
		return
	}
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
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilityTemplateWriteMetadataKnownKey, []byte("true"))...)
	}
}

func (r *observabilityTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestedID := model.ID.ValueString()
	result, err := r.Details().Client.GetTemplate(ctx, requestedID, nil)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := observabilityTemplateFromResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading template", err.Error())
		return
	}
	if record.ID != requestedID {
		resp.Diagnostics.AddError(
			"Error reading template",
			fmt.Sprintf("Template API returned record %q when reading Template %q.", record.ID, requestedID),
		)
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
	var knownMetadata []byte
	if req.Private != nil {
		var privateDiags diag.Diagnostics
		knownMetadata, privateDiags = req.Private.GetKey(ctx, observabilityTemplateWriteMetadataKnownKey)
		resp.Diagnostics.Append(privateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(knownMetadata) == 0 && model.Metadata == nil {
		resp.Diagnostics.AddError(observabilityTemplateUnsafeUpdateSummary, observabilityTemplateUnsafeUpdateDetail)
		return
	}

	write, diags := observabilityTemplateWrite(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.UpdateTemplate(ctx, model.ID.ValueString(), write)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.Diagnostics.AddError("Error updating template", fmt.Sprintf("Template %q was not found during update; retry after refreshing the Terraform state.", model.ID.ValueString()))
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if _, err := observabilityTemplateFromResult(result); err != nil {
		resp.Diagnostics.AddError("Error updating template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilityTemplateWriteMetadataKnownKey, []byte("true"))...)
	}
}

func (r *observabilityTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := model.ID.ValueString()
	result, lookupErr := r.Details().Client.GetTemplate(ctx, id, nil)
	if lookupErr == nil && result != nil && result.Data != nil && len(result.Data.DirectoryEntries) > 0 {
		resp.Diagnostics.AddWarning(
			"Deleting Template with Directory memberships",
			fmt.Sprintf(
				"The Template is referenced by %d Directory entry(s). Deleting it can leave dangling memberships in those Directories; update them after deletion.",
				len(result.Data.DirectoryEntries),
			),
		)
	}

	err := r.Details().Client.DeleteTemplate(ctx, id)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func validateObservabilityTitle(resp *resource.ValidateConfigResponse, titlePath path.Path, title types.String) {
	if title.IsUnknown() {
		return
	}
	if title.IsNull() || strings.TrimSpace(title.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(titlePath, "Missing required value", "title must contain at least one non-whitespace character")
	}
}

func observabilityTemplateFromResult(result *template.Result) (*template.Template, error) {
	if result == nil || result.Data == nil {
		return nil, errors.New("template API returned no template record")
	}
	if result.Data.ID == "" {
		return nil, errors.New("template API returned a template record without an ID")
	}
	return result.Data, nil
}

func observabilityTemplateWrite(ctx context.Context, model observabilityTemplateModel) (*template.CreateUpdateTemplateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	if model.RootElement.IsNull() || model.RootElement.IsUnknown() || model.RootElement.ValueString() == "" {
		diags.AddError("Missing template root element", "root_element must be a non-empty value.")
		return nil, diags
	}
	if err := validateObservabilityTemplateSpec(model.Spec.ValueString()); err != nil {
		diags.AddError("Invalid template specification", err.Error())
		return nil, diags
	}

	var imports []string
	var datasource *template.Datasource
	if model.Metadata != nil {
		if !model.Metadata.Imports.IsNull() && !model.Metadata.Imports.IsUnknown() {
			var ids []string
			diags.Append(model.Metadata.Imports.ElementsAs(ctx, &ids, false)...)
			if diags.HasError() {
				return nil, diags
			}
			imports = make([]string, len(ids))
			for i, id := range ids {
				if !observabilityTemplateIDValid(id) {
					diags.AddAttributeError(path.Root("metadata").AtName("imports").AtListIndex(i), "Invalid Template ID", "Use a Template ID without an API path.")
					return nil, diags
				}
				imports[i] = observabilityTemplateReference(id)
			}
		}

		if model.Metadata.Datasource != nil {
			datasource = &template.Datasource{
				Type:        template.DatasourceType(model.Metadata.Datasource.Type.ValueString()),
				ProgramText: model.Metadata.Datasource.ProgramText.ValueString(),
				SLOID:       model.Metadata.Datasource.SLOID.ValueString(),
			}
		}
	}

	rootElement := template.RootElement(model.RootElement.ValueString())
	return &template.CreateUpdateTemplateRequest{
		Type:  observabilityTemplateRecordType,
		Title: model.Title.ValueString(),
		Spec:  json.RawMessage(model.Spec.ValueString()),
		Metadata: template.WriteMetadata{
			RootElement: &rootElement,
			Imports:     imports,
			Datasource:  datasource,
		},
	}, diags
}

func observabilityTemplateModelFromRecord(prior observabilityTemplateModel, record *template.Template) (observabilityTemplateModel, error) {
	if record.Metadata == nil || record.Metadata.RootElement == nil {
		return observabilityTemplateModel{}, errors.New("template API returned a template record without root element metadata")
	}
	if err := validateObservabilityTemplateSpec(string(record.Spec)); err != nil {
		return observabilityTemplateModel{}, fmt.Errorf("template API returned an invalid specification: %w", err)
	}

	return observabilityTemplateModel{
		ID:          types.StringValue(record.ID),
		Title:       types.StringValue(record.Title),
		RootElement: types.StringValue(string(*record.Metadata.RootElement)),
		Spec:        jsontypes.NewNormalizedValue(string(record.Spec)),
		Metadata:    prior.Metadata,
	}, nil
}

func validateObservabilityTemplateSpec(raw string) error {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("spec must be valid JSON: %w", err)
	}
	if _, ok := value.(map[string]any); !ok {
		return errors.New("spec must be a JSON object")
	}
	return nil
}

func observabilityTemplateIDListValidators() []validator.List {
	return []validator.List{
		listvalidator.NoNullValues(),
		listvalidator.ValueStringsAre(
			stringvalidator.RegexMatches(regexp.MustCompile(`^[^/]+$`), "use a Template ID without an API path"),
			stringvalidator.NoneOf(".", ".."),
		),
		listvalidator.UniqueValues(),
	}
}

func observabilityTemplateIDValid(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.Contains(id, "/")
}

func observabilityTemplateReference(id string) string {
	return observabilityTemplateReferencePrefix + id
}

func observabilityTemplateIDFromReference(reference string) (string, bool) {
	id, ok := strings.CutPrefix(reference, observabilityTemplateReferencePrefix)
	return id, ok && observabilityTemplateIDValid(id)
}
