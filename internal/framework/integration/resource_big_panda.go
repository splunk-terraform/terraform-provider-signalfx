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

type ResourceBigPanda struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceBigPandaModel struct {
	Id                            types.String `tfsdk:"id"`
	Enabled                       types.Bool   `tfsdk:"enabled"`
	Name                          types.String `tfsdk:"name"`
	AppKey                        types.String `tfsdk:"app_key"`
	Token                         types.String `tfsdk:"token"`
	AlertTriggeredPayloadTemplate types.String `tfsdk:"alert_triggered_payload_template"`
	AlertResolvedPayloadTemplate  types.String `tfsdk:"alert_resolved_payload_template"`
}

var (
	_ resource.Resource                = &ResourceBigPanda{}
	_ resource.ResourceWithConfigure   = &ResourceBigPanda{}
	_ resource.ResourceWithImportState = &ResourceBigPanda{}
)

func NewResourceBigPanda() resource.Resource {
	return &ResourceBigPanda{}
}

func (bp *ResourceBigPanda) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_big_panda_integration"
}

func (bp *ResourceBigPanda) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this resource to manage a BigPanda integration.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Enables or disables the BigPanda integration.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Used to provide a human-readable name for the BigPanda integration.",
			},
			"app_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Application key you get from BigPanda.",
			},
			"token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Token you get from BigPanda.",
			},
			"alert_triggered_payload_template": schema.StringAttribute{
				Optional:    true,
				Description: "A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a triggered notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.",
			},
			"alert_resolved_payload_template": schema.StringAttribute{
				Optional:    true,
				Description: "A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a resolved notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.",
			},
		},
	}
}

func (bp *ResourceBigPanda) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceBigPandaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := bp.Details().Client.CreateBigPandaIntegration(ctx, model.toIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	model.updateFromIntegration(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (bp *ResourceBigPanda) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceBigPandaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := bp.Details().Client.GetBigPandaIntegration(ctx, model.Id.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateFromIntegration(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (bp *ResourceBigPanda) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceBigPandaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := bp.Details().Client.UpdateBigPandaIntegration(ctx, model.Id.ValueString(), model.toIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateFromIntegration(details)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (bp *ResourceBigPanda) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceBigPandaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := bp.Details().Client.DeleteBigPandaIntegration(ctx, model.Id.ValueString())
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func (model resourceBigPandaModel) toIntegration() *integration.BigPandaIntegration {
	details := &integration.BigPandaIntegration{
		Type:    integration.BIG_PANDA,
		Enabled: model.Enabled.ValueBool(),
		Name:    model.Name.ValueString(),
		AppKey:  model.AppKey.ValueString(),
		Token:   model.Token.ValueString(),
	}

	if !model.AlertTriggeredPayloadTemplate.IsNull() && !model.AlertTriggeredPayloadTemplate.IsUnknown() {
		details.AlertTriggeredPayloadTemplate = model.AlertTriggeredPayloadTemplate.ValueString()
	}
	if !model.AlertResolvedPayloadTemplate.IsNull() && !model.AlertResolvedPayloadTemplate.IsUnknown() {
		details.AlertResolvedPayloadTemplate = model.AlertResolvedPayloadTemplate.ValueString()
	}

	return details
}

func (model *resourceBigPandaModel) updateFromIntegration(details *integration.BigPandaIntegration) {
	model.Id = types.StringValue(details.Id)
	model.Enabled = types.BoolValue(details.Enabled)
	model.Name = types.StringValue(details.Name)
	model.AlertTriggeredPayloadTemplate = optionalStringValue(details.AlertTriggeredPayloadTemplate)
	model.AlertResolvedPayloadTemplate = optionalStringValue(details.AlertResolvedPayloadTemplate)
}

func optionalStringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
