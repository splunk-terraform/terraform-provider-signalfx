// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	TemplateAPIPath = "/v2/template"
)

type ResourceDashifyTemplate struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter

	net *http.Client
}

type resourceDashifyTemplateModel struct {
	Id               types.String `tfsdk:"id"`
	TemplateContents types.String `tfsdk:"template_contents"`
}

var (
	_ resource.Resource                = (*ResourceDashifyTemplate)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceDashifyTemplate)(nil)
	_ resource.ResourceWithImportState = (*ResourceDashifyTemplate)(nil)
)

func NewResourceDashifyTemplate() resource.Resource {
	return &ResourceDashifyTemplate{
		net: &http.Client{
			// TODO(smarciniak): Move this to be part of the official SDK once GA.
		},
	}
}

func (r *ResourceDashifyTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashify_template"
}

func (r *ResourceDashifyTemplate) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Dashify templates for modern dashboards in Splunk Observability Cloud",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"template_contents": schema.StringAttribute{
				Required:    true,
				Description: "JSON contents of the dashify template. Must be valid JSON containing the template structure with metadata and spec sections.",
			},
		},
	}
}

func (r *ResourceDashifyTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate JSON
	templateContents := model.TemplateContents.ValueString()
	var js any
	err := json.Unmarshal([]byte(templateContents), &js)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid JSON",
			fmt.Sprintf("template_contents must be valid JSON: %s", err.Error()),
		)
		return
	}

	// Make API request
	httpResp, err := r.doRequest(ctx, http.MethodPost, path.Join(TemplateAPIPath), js)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Template",
			fmt.Sprintf("Could not create dashify template: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	// Parse response to get the template ID
	// The API returns: {"data": {"id": "...", ...}, "errors": [], "includes": []}
	var result map[string]any
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("Could not parse response: %s", err.Error()),
		)
		return
	}

	// Extract the data object
	data, ok := result["data"].(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("data field not found in response: %v", result),
		)
		return
	}

	// Extract the ID from data.id
	templateID, ok := data["id"].(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("template ID not found in response data: %v", result),
		)
		return
	}

	model.Id = types.StringValue(templateID)
	tflog.Debug(ctx, "Created Dashify Template", map[string]any{"id": templateID})

	// Don't read back - keep the user's original input to avoid inconsistencies
	// The API adds default fields (like metadata.imports, type) that weren't in the input
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ResourceDashifyTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readTemplate(ctx, &model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.Id.IsNull() {
		// Template was deleted
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ResourceDashifyTemplate) readTemplate(ctx context.Context, model *resourceDashifyTemplateModel, diags *diag.Diagnostics) {
	resp, err := r.doRequest(ctx, http.MethodGet, path.Join(TemplateAPIPath, model.Id.ValueString()), nil)
	if err != nil {
		diags.AddError(
			"Error Making API Request",
			fmt.Sprintf("Could not make API request: %s", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	// Parse response - GET may return data in same format as POST
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		diags.AddError(
			"Error Parsing Response",
			fmt.Sprintf("Could not parse response: %s", err.Error()),
		)
		return
	}

	// Check if response has a "data" wrapper
	var templateData map[string]any
	if data, ok := result["data"].(map[string]any); ok {
		// Response is wrapped in {"data": {...}}
		templateData = data
	} else {
		// Response is direct template data
		templateData = result
	}

	// Filter out API-generated fields to maintain consistency with input
	// Only keep the fields that users provide: metadata, spec, title, type (if user-provided)
	filteredData := make(map[string]any)

	// These are the fields users provide in their template
	userProvidedFields := []string{"metadata", "spec", "title", "type"}
	for _, field := range userProvidedFields {
		if val, exists := templateData[field]; exists {
			filteredData[field] = val
		}
	}

	// Set the template contents - marshal back to JSON (without API metadata)
	templateJSON, err := json.Marshal(filteredData)
	if err != nil {
		diags.AddError(
			"Error Marshaling Template",
			fmt.Sprintf("Could not marshal template: %s", err.Error()),
		)
		return
	}

	model.TemplateContents = types.StringValue(string(templateJSON))

	tflog.Debug(ctx, "Read Dashify Template", map[string]any{"id": model.Id.ValueString()})
}

func (r *ResourceDashifyTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate JSON
	templateContents := model.TemplateContents.ValueString()
	var js any
	if err := json.Unmarshal([]byte(templateContents), &js); err != nil {
		resp.Diagnostics.AddError(
			"Invalid JSON",
			fmt.Sprintf("template_contents must be valid JSON: %s", err.Error()),
		)
		return
	}

	httpResp, err := r.doRequest(ctx, http.MethodPut, path.Join(TemplateAPIPath, model.Id.ValueString()), js)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Template",
			fmt.Sprintf("Could not update dashify template: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	tflog.Debug(ctx, "Updated Dashify Template", tfext.NewLogFields().Field("id", model.Id.ValueString()))

	// Don't read back - keep the user's original input to avoid inconsistencies
	// The API adds default fields (like metadata.imports, type) that weren't in the input
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ResourceDashifyTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.doRequest(ctx, http.MethodDelete, path.Join(TemplateAPIPath, model.Id.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Template",
			fmt.Sprintf("Could not delete dashify template: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	tflog.Debug(ctx, "Deleted Dashify Template", tfext.NewLogFields().Field("id", model.Id.ValueString()))
}

func (r *ResourceDashifyTemplate) doRequest(ctx context.Context, method string, path string, body any) (*http.Response, error) {
	u, err := url.ParseRequestURI(r.Details().APIURL)
	if err != nil {
		return nil, err
	}
	u = u.JoinPath(path)

	var content io.Reader = http.NoBody
	if body != nil {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
		content = buf
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), content)
	if err != nil {
		return nil, err
	}

	details := r.Details()
	req.Header.Set("X-Sf-Token", details.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.net.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return resp, nil
	default:
		content, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(content))
	}
}
