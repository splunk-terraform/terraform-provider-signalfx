// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package charts implements the reusable Observability Chart Template resource.
// Curated YAML definitions drive its Terraform attributes and chart documents.
package charts

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityChartResource struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
	engine      *interpreter.Engine
	schemaError error
}

var (
	_ resource.Resource                   = (*observabilityChartResource)(nil)
	_ resource.ResourceWithConfigure      = (*observabilityChartResource)(nil)
	_ resource.ResourceWithValidateConfig = (*observabilityChartResource)(nil)
)

func NewResourceObservabilityChart() resource.Resource {
	engine, err := interpreter.Default()
	return &observabilityChartResource{engine: engine, schemaError: err}
}

func (r *observabilityChartResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_chart"
}

func (r *observabilityChartResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.schemaError != nil {
		resp.Diagnostics.AddError("Invalid chart schema", r.schemaError.Error())
		return
	}
	attributes := map[string]schema.Attribute{
		"id":    fwshared.ResourceIDAttribute(),
		"title": schema.StringAttribute{Required: true, Description: "Template record title. The chart display title is configured separately."},
	}
	for _, chart := range r.engine.Charts() {
		nested, err := r.engine.Attributes(chart.Name)
		if err != nil {
			resp.Diagnostics.AddError("Invalid chart schema", err.Error())
			return
		}
		attributes[chart.Name] = schema.SingleNestedAttribute{Optional: true, MarkdownDescription: chart.Description, Attributes: nested}
	}
	resp.Schema = schema.Schema{
		Description: "Manages a reusable Observability Chart Template. Configure exactly one chart type. Updates replace the complete saved chart and discard unsupported fields. Reference its ID from a Dashboard panel; Directory placement is managed by signalfx_observability_directory.",
		Attributes:  attributes,
	}
}

func (r *observabilityChartResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	model, diags := r.model(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	fwdashify.ValidateObservabilityTitle(resp, path.Root("title"), model.Title)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.engine.Validate(ctx, model.Name, model.Content, false)...)
	}
}

func (r *observabilityChartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	model, diags := r.model(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	write, diags := r.write(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.CreateTemplate(ctx, write)
	if isObservabilityChartNotFound(err) {
		resp.Diagnostics.AddError("Error creating chart", "The Template API endpoint was not found.")
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	parsed, diags := r.fromResult(ctx, result, "", model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if parsed.Name != model.Name {
		resp.Diagnostics.AddError("Invalid Chart API response", fmt.Sprintf("Template API returned chart type %q for configured %q.", parsed.Name, model.Name))
		return
	}
	model.ID = parsed.ID
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
}

func (r *observabilityChartResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() || !validateObservabilityChartID(id, &resp.Diagnostics) {
		return
	}
	result, err := r.Details().Client.GetTemplate(ctx, id.ValueString(), nil)
	if isObservabilityChartNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	// Imported state has only an ID; absence of prior chart content is expected.
	prior, _ := r.model(ctx, req.State)
	model, diags := r.fromResult(ctx, result, id.ValueString(), prior)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
	}
}

func (r *observabilityChartResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	model, diags := r.model(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &model.ID)...)
	if resp.Diagnostics.HasError() || !validateObservabilityChartID(model.ID, &resp.Diagnostics) {
		return
	}
	write, diags := r.write(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.Details().Client.UpdateTemplate(ctx, model.ID.ValueString(), write)
	if isObservabilityChartNotFound(err) {
		resp.Diagnostics.AddError("Error updating chart", fmt.Sprintf("Chart %q was not found during update; retry after refreshing the Terraform state.", model.ID.ValueString()))
		return
	}
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	parsed, diags := r.fromResult(ctx, result, model.ID.ValueString(), model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if parsed.Name != model.Name {
		resp.Diagnostics.AddError("Invalid Chart API response", fmt.Sprintf("Template API returned chart type %q for configured %q.", parsed.Name, model.Name))
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, model)...)
}

func (r *observabilityChartResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() || !validateObservabilityChartID(id, &resp.Diagnostics) {
		return
	}
	err := r.Details().Client.DeleteTemplate(ctx, id.ValueString())
	if !isObservabilityChartNotFound(err) {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
	}
}

func validateObservabilityChartID(id types.String, diags *diag.Diagnostics) bool {
	if id.IsNull() || id.IsUnknown() || !fwdashify.ObservabilityTemplateIDValid(id.ValueString()) {
		diags.AddAttributeError(path.Root("id"), "Invalid Template ID", "Use a bare Template ID without an API path.")
		return false
	}
	return true
}

func isObservabilityChartNotFound(err error) bool {
	responseErr, ok := signalfx.AsResponseError(err)
	return ok && responseErr.Code() == http.StatusNotFound
}
