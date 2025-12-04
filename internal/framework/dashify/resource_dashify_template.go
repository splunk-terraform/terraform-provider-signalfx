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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

const (
	TemplateAPIPath = "/v2/template"
)

type ResourceDashifyTemplate struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceDashifyTemplateModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	TemplateContents types.String `tfsdk:"template_contents"`
}

var (
	_ resource.Resource                = &ResourceDashifyTemplate{}
	_ resource.ResourceWithConfigure   = &ResourceDashifyTemplate{}
	_ resource.ResourceWithImportState = &ResourceDashifyTemplate{}
)

func NewResourceDashifyTemplate() resource.Resource {
	return &ResourceDashifyTemplate{}
}

func (r *ResourceDashifyTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashify_template"
}

func (r *ResourceDashifyTemplate) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Dashify templates for modern dashboards in Splunk Observability Cloud",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the dashify template",
			},
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
	var js interface{}
	if err := json.Unmarshal([]byte(templateContents), &js); err != nil {
		resp.Diagnostics.AddError(
			"Invalid JSON",
			fmt.Sprintf("template_contents must be valid JSON: %s", err.Error()),
		)
		return
	}

	// Make API request
	details := r.Details()
	url := fmt.Sprintf("%s%s", details.APIURL, TemplateAPIPath)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(templateContents)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating HTTP Request",
			fmt.Sprintf("Could not create HTTP request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-SF-TOKEN", details.AuthToken)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Making API Request",
			fmt.Sprintf("Could not make API request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Response",
			fmt.Sprintf("Could not read response body: %s", err.Error()),
		)
		return
	}

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError(
			"Error Creating Template",
			fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)),
		)
		return
	}

	// Parse response to get the template ID
	// The API returns: {"data": {"id": "...", ...}, "errors": [], "includes": []}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("Could not parse response: %s", err.Error()),
		)
		return
	}

	// Extract the data object
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("data field not found in response: %s", string(body)),
		)
		return
	}

	// Extract the ID from data.id
	templateID, ok := data["id"].(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Error Parsing Response",
			fmt.Sprintf("template ID not found in response data: %s", string(body)),
		)
		return
	}

	model.Id = types.StringValue(templateID)
	tflog.Debug(ctx, "Created Dashify Template", map[string]interface{}{"id": templateID})

	// Read back the template to ensure state is up to date
	r.readTemplate(ctx, &model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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
	details := r.Details()
	templateID := model.Id.ValueString()
	url := fmt.Sprintf("%s%s/%s", details.APIURL, TemplateAPIPath, templateID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		diags.AddError(
			"Error Creating HTTP Request",
			fmt.Sprintf("Could not create HTTP request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("X-SF-TOKEN", details.AuthToken)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		diags.AddError(
			"Error Making API Request",
			fmt.Sprintf("Could not make API request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "Dashify Template not found", map[string]interface{}{"id": templateID})
		model.Id = types.StringNull()
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		diags.AddError(
			"Error Reading Response",
			fmt.Sprintf("Could not read response body: %s", err.Error()),
		)
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		diags.AddError(
			"Error Reading Template",
			fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)),
		)
		return
	}

	// Parse response - GET may return data in same format as POST
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		diags.AddError(
			"Error Parsing Response",
			fmt.Sprintf("Could not parse response: %s", err.Error()),
		)
		return
	}

	// Check if response has a "data" wrapper
	var templateData map[string]interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		// Response is wrapped in {"data": {...}}
		templateData = data
	} else {
		// Response is direct template data
		templateData = result
	}

	// Set the template contents - marshal back to JSON
	templateJSON, err := json.Marshal(templateData)
	if err != nil {
		diags.AddError(
			"Error Marshaling Template",
			fmt.Sprintf("Could not marshal template: %s", err.Error()),
		)
		return
	}

	model.TemplateContents = types.StringValue(string(templateJSON))

	// Set name/title if present
	if title, ok := templateData["title"].(string); ok {
		model.Name = types.StringValue(title)
	}

	tflog.Debug(ctx, "Read Dashify Template", map[string]interface{}{"id": templateID})
}

func (r *ResourceDashifyTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate JSON
	templateContents := model.TemplateContents.ValueString()
	var js interface{}
	if err := json.Unmarshal([]byte(templateContents), &js); err != nil {
		resp.Diagnostics.AddError(
			"Invalid JSON",
			fmt.Sprintf("template_contents must be valid JSON: %s", err.Error()),
		)
		return
	}

	// Make API request to update
	details := r.Details()
	templateID := model.Id.ValueString()
	url := fmt.Sprintf("%s%s/%s", details.APIURL, TemplateAPIPath, templateID)

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer([]byte(templateContents)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating HTTP Request",
			fmt.Sprintf("Could not create HTTP request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-SF-TOKEN", details.AuthToken)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Making API Request",
			fmt.Sprintf("Could not make API request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Response",
			fmt.Sprintf("Could not read response body: %s", err.Error()),
		)
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Error Updating Template",
			fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)),
		)
		return
	}

	tflog.Debug(ctx, "Updated Dashify Template", map[string]interface{}{"id": templateID})

	// Read back the template to ensure state is up to date
	r.readTemplate(ctx, &model, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *ResourceDashifyTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceDashifyTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details := r.Details()
	templateID := model.Id.ValueString()
	url := fmt.Sprintf("%s%s/%s", details.APIURL, TemplateAPIPath, templateID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating HTTP Request",
			fmt.Sprintf("Could not create HTTP request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("X-SF-TOKEN", details.AuthToken)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Making API Request",
			fmt.Sprintf("Could not make API request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError(
			"Error Deleting Template",
			fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)),
		)
		return
	}

	tflog.Debug(ctx, "Deleted Dashify Template", map[string]interface{}{"id": templateID})
}
