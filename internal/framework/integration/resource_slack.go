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
)

type ResourceSlack struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceSlackModel struct {
	integrationModel
	WebhookURL types.String `tfsdk:"webhook_url"`
}

var (
	_ resource.Resource                = (*ResourceSlack)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceSlack)(nil)
	_ resource.ResourceWithImportState = (*ResourceSlack)(nil)
)

func NewResourceSlack() resource.Resource { return &ResourceSlack{} }

func (slack *ResourceSlack) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slack_integration"
}

func (slack *ResourceSlack) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["webhook_url"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "Slack incoming webhook URL used to send notifications.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages a Slack webhook integration in Splunk Observability Cloud.",
		Attributes:  attributes,
	}
}

func (slack *ResourceSlack) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceSlackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := slack.Details().Client.CreateSlackIntegration(ctx, model.slackIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (slack *ResourceSlack) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceSlackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := slack.Details().Client.GetSlackIntegration(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model.updateFromAPI(details, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (slack *ResourceSlack) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceSlackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := slack.Details().Client.UpdateSlackIntegration(ctx, model.ID.ValueString(), model.slackIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (slack *ResourceSlack) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceSlackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, slack.Details().Client.DeleteSlackIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceSlackModel) slackIntegration() *integration.SlackIntegration {
	return &integration.SlackIntegration{
		Type:       "Slack",
		Name:       model.Name.ValueString(),
		Enabled:    model.Enabled.ValueBool(),
		WebhookUrl: model.WebhookURL.ValueString(),
	}
}

func (model *resourceSlackModel) updateFromAPI(details *integration.SlackIntegration, updateID bool) {
	if details == nil {
		return
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	// The API omits webhook_url. Retain the configured value in state.
}
