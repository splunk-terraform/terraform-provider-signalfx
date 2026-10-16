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
	"sort"
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

func (r *observabilityTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability Template, such as a chart.",
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
				Description: "Root element, such as Chart or Dashboard. Changing it replaces the Template.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"spec": schema.StringAttribute{
				Required:    true,
				Description: "Template content as a JSON object.",
				CustomType:  jsontypes.NormalizedType{},
				PlanModifiers: []planmodifier.String{
					observabilityTemplateSpecSemanticPlanModifier{},
				},
			},
			"metadata": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "API metadata. Templates saved in the dashboard UI usually do not need this.",
				Attributes: map[string]schema.Attribute{
					"imports": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Override the Template IDs found in spec.",
						Validators:  observabilityTemplateIDListValidators(),
					},
					"datasource": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Separate datasource metadata used by some API clients. The API cannot read it back.",
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

// Keep the stored JSON when a config change only alters whitespace or key
// order. Without this, importing the API's compact JSON and using a readable
// heredoc in Terraform produces a spurious in-place update.
type observabilityTemplateSpecSemanticPlanModifier struct{}

func (observabilityTemplateSpecSemanticPlanModifier) Description(context.Context) string {
	return "Preserves the existing specification when the JSON content is unchanged."
}

func (m observabilityTemplateSpecSemanticPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (observabilityTemplateSpecSemanticPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	equal, diags := jsontypes.NewNormalizedValue(req.StateValue.ValueString()).StringSemanticEquals(
		ctx, jsontypes.NewNormalizedValue(req.PlanValue.ValueString()),
	)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() && equal {
		resp.PlanValue = req.StateValue
	}
}

func (r *observabilityTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ValidateObservabilityTitle(resp, path.Root("title"), model.Title)
	if !model.Spec.IsNull() && !model.Spec.IsUnknown() {
		if err := validateObservabilityTemplateSpec(model.Spec.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("spec"), "Invalid template specification", err.Error())
		} else if model.Metadata == nil || model.Metadata.Imports.IsNull() {
			if _, err := observabilityTemplateImportsFromSpec(model.Spec.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("spec"), "Invalid template import", err.Error())
			}
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
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.Diagnostics.AddError("Error creating template", "The Template API endpoint was not found.")
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := ObservabilityTemplateFromResult(result)
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

	requestedID := model.ID.ValueString()
	result, err := r.Details().Client.GetTemplate(ctx, requestedID, nil)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	record, err := ObservabilityTemplateFromResult(result)
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
	if _, err := ObservabilityTemplateFromResult(result); err != nil {
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

	id := model.ID.ValueString()
	err := r.Details().Client.DeleteTemplate(ctx, id)
	if responseErr, ok := signalfx.AsResponseError(err); ok && responseErr.Code() == http.StatusNotFound {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
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
	if model.Metadata == nil || model.Metadata.Imports.IsNull() {
		var err error
		imports, err = observabilityTemplateImportsFromSpec(model.Spec.ValueString())
		if err != nil {
			diags.AddAttributeError(path.Root("spec"), "Invalid template import", err.Error())
			return nil, diags
		}
	} else {
		if model.Metadata.Imports.IsUnknown() {
			diags.AddAttributeError(path.Root("metadata").AtName("imports"), "Unknown template imports", "metadata.imports must be known before writing a Template.")
			return nil, diags
		}
		var ids []string
		diags.Append(model.Metadata.Imports.ElementsAs(ctx, &ids, false)...)
		if diags.HasError() {
			return nil, diags
		}
		imports = make([]string, len(ids))
		for i, id := range ids {
			if !ObservabilityTemplateIDValid(id) {
				diags.AddAttributeError(path.Root("metadata").AtName("imports").AtListIndex(i), "Invalid Template ID", "Use a Template ID without an API path.")
				return nil, diags
			}
			imports[i] = ObservabilityTemplateReference(id)
		}
	}

	if model.Metadata != nil {
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
		Type:  ObservabilityTemplateRecordType,
		Title: model.Title.ValueString(),
		Spec:  json.RawMessage(model.Spec.ValueString()),
		Metadata: template.WriteMetadata{
			RootElement: &rootElement,
			Imports:     imports,
			Datasource:  datasource,
		},
	}, diags
}

// The Template API's read-side imports are empty on a default GET and can be
// transitive on an expanded GET. Derive the write list from the direct imports
// in the root spec.
func observabilityTemplateImportsFromSpec(raw string) ([]string, error) {
	var spec map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(spec))
	for key := range spec {
		if strings.HasPrefix(key, observabilityTemplateImportPrefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	imports := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		var reference string
		if err := json.Unmarshal(spec[key], &reference); err != nil {
			return nil, fmt.Errorf("%q must contain a Template API reference: %w", key, err)
		}
		if _, ok := ObservabilityTemplateIDFromReference(reference); !ok {
			return nil, fmt.Errorf("%q must reference a Template at %s<ID>, got %q", key, observabilityTemplateReferencePrefix, reference)
		}
		if _, exists := seen[reference]; !exists {
			imports = append(imports, reference)
			seen[reference] = struct{}{}
		}
	}
	return imports, nil
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
