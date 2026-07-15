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

type ResourcePagerDuty struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourcePagerDutyModel struct {
	integrationModel
	APIKey types.String `tfsdk:"api_key"`
}

var (
	_ resource.Resource                = (*ResourcePagerDuty)(nil)
	_ resource.ResourceWithConfigure   = (*ResourcePagerDuty)(nil)
	_ resource.ResourceWithImportState = (*ResourcePagerDuty)(nil)
)

func NewResourcePagerDuty() resource.Resource {
	return &ResourcePagerDuty{}
}

func (pagerDuty *ResourcePagerDuty) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pagerduty_integration"
}

func (pagerDuty *ResourcePagerDuty) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["api_key"] = schema.StringAttribute{
		Optional:    true,
		Sensitive:   true,
		Description: "PagerDuty integration key used to send alerts. PagerDuty refers to this value as the integration key.",
	}

	resp.Schema = schema.Schema{
		Description: "Manages a PagerDuty notification integration in Splunk Observability Cloud.",
		Attributes:  attributes,
	}
}

func (pagerDuty *ResourcePagerDuty) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourcePagerDutyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := pagerDuty.Details().Client.CreatePagerDutyIntegration(ctx, model.pagerDutyIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}

	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (pagerDuty *ResourcePagerDuty) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourcePagerDutyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := pagerDuty.Details().Client.GetPagerDutyIntegration(ctx, model.ID.ValueString())
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

func (pagerDuty *ResourcePagerDuty) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourcePagerDutyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := pagerDuty.Details().Client.UpdatePagerDutyIntegration(
		ctx,
		model.ID.ValueString(),
		model.pagerDutyIntegration(),
	)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateFromAPI(details, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (pagerDuty *ResourcePagerDuty) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourcePagerDutyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := pagerDuty.Details().Client.DeletePagerDutyIntegration(ctx, model.ID.ValueString())
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func (model resourcePagerDutyModel) pagerDutyIntegration() *integration.PagerDutyIntegration {
	return &integration.PagerDutyIntegration{
		Type:    "PagerDuty",
		Name:    model.Name.ValueString(),
		Enabled: model.Enabled.ValueBool(),
		ApiKey:  model.APIKey.ValueString(),
	}
}

func (model *resourcePagerDutyModel) updateFromAPI(details *integration.PagerDutyIntegration, updateID bool) {
	if details == nil {
		return
	}

	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	// The API intentionally omits api_key. Retain its configured value so
	// every refresh remains stable.
}
