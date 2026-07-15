// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceOpsgenie struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceOpsgenieModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
	APIKey  types.String `tfsdk:"api_key"`
	APIURL  types.String `tfsdk:"api_url"`
}

var (
	_ resource.Resource                = (*ResourceOpsgenie)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceOpsgenie)(nil)
	_ resource.ResourceWithImportState = (*ResourceOpsgenie)(nil)
)

func NewResourceOpsgenie() resource.Resource {
	return &ResourceOpsgenie{}
}

func (opsgenie *ResourceOpsgenie) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_opsgenie_integration"
}

func (opsgenie *ResourceOpsgenie) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Opsgenie notification integration in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name of the Opsgenie integration.",
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether the Opsgenie integration is enabled.",
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Opsgenie API key used to send alerts.",
			},
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "Opsgenie API URL. Use the regional URL required by your Opsgenie account.",
			},
		},
	}
}

func (opsgenie *ResourceOpsgenie) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceOpsgenieModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := opsgenie.Details().Client.CreateOpsgenieIntegration(ctx, model.opsgenieIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}

	model.updateFromAPI(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (opsgenie *ResourceOpsgenie) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceOpsgenieModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := opsgenie.Details().Client.GetOpsgenieIntegration(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateFromAPI(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (opsgenie *ResourceOpsgenie) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceOpsgenieModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := opsgenie.Details().Client.UpdateOpsgenieIntegration(
		ctx,
		model.ID.ValueString(),
		model.opsgenieIntegration(),
	)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateFromAPI(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (opsgenie *ResourceOpsgenie) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceOpsgenieModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := opsgenie.Details().Client.DeleteOpsgenieIntegration(ctx, model.ID.ValueString())
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func (model resourceOpsgenieModel) opsgenieIntegration() *integration.OpsgenieIntegration {
	return &integration.OpsgenieIntegration{
		Type:    "Opsgenie",
		Name:    model.Name.ValueString(),
		Enabled: model.Enabled.ValueBool(),
		ApiKey:  model.APIKey.ValueString(),
		ApiUrl:  model.APIURL.ValueString(),
	}
}

func (model *resourceOpsgenieModel) updateFromAPI(details *integration.OpsgenieIntegration) {
	if details == nil {
		return
	}

	model.ID = types.StringValue(details.Id)
	model.Name = types.StringValue(details.Name)
	model.Enabled = types.BoolValue(details.Enabled)
	// The API intentionally omits api_key and api_url. Retain their configured
	// values so every refresh remains stable.
}
