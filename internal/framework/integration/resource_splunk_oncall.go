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

type ResourceSplunkOncall struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceSplunkOnCallModel struct {
	Id      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Name    types.String `tfsdk:"name"`
	PostURL types.String `tfsdk:"post_url"`
}

var (
	_ resource.Resource                = &ResourceSplunkOncall{}
	_ resource.ResourceWithConfigure   = &ResourceSplunkOncall{}
	_ resource.ResourceWithImportState = &ResourceSplunkOncall{}
)

func NewResourceSplunkOncall() resource.Resource {
	return &ResourceSplunkOncall{}
}

func (oncall *ResourceSplunkOncall) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_splunk_oncall"
}

func (oncall *ResourceSplunkOncall) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this resource to manage a Splunk Oncall Integration",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Enables or disables the Splunk Oncall integration.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Used to provide a human-readable name for the Splunk Oncall integration.",
			},
			"post_url": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "This is the Splunk OnCall integration URL.",
			},
		},
	}
}

func (oncall *ResourceSplunkOncall) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceSplunkOnCallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := oncall.Details().Client.CreateVictorOpsIntegration(
		ctx,
		&integration.VictorOpsIntegration{
			// Internally this still uses the VictorOps details
			// but the API has been rebranded to Splunk Oncall.
			Type:    integration.VICTOR_OPS,
			Enabled: model.Enabled.ValueBool(),
			Name:    model.Name.ValueString(),
			PostUrl: model.PostURL.ValueString(),
		},
	)

	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	model.Id = types.StringValue(details.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (oncall *ResourceSplunkOncall) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceSplunkOnCallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := oncall.Details().Client.GetVictorOpsIntegration(
		ctx,
		model.Id.ValueString(),
	)

	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.Enabled = types.BoolValue(details.Enabled)
	model.Name = types.StringValue(details.Name)
	model.PostURL = types.StringValue(details.PostUrl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (oncall *ResourceSplunkOncall) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceSplunkOnCallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := oncall.Details().Client.UpdateVictorOpsIntegration(
		ctx,
		model.Id.ValueString(),
		&integration.VictorOpsIntegration{
			Type:    integration.VICTOR_OPS,
			Enabled: model.Enabled.ValueBool(),
			Name:    model.Name.ValueString(),
			PostUrl: model.PostURL.ValueString(),
		},
	)

	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.Enabled = types.BoolValue(details.Enabled)
	model.Name = types.StringValue(details.Name)
	model.PostURL = types.StringValue(details.PostUrl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (oncall *ResourceSplunkOncall) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceSplunkOnCallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := oncall.Details().Client.DeleteVictorOpsIntegration(
		ctx,
		model.Id.ValueString(),
	)

	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}
