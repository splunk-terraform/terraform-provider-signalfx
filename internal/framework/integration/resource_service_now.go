// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

const (
	serviceNowTypeIncident = "Incident"
	serviceNowTypeProblem  = "Problem"
)

type ResourceServiceNow struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceServiceNowModel struct {
	integrationModel
	Username                      types.String `tfsdk:"username"`
	Password                      types.String `tfsdk:"password"`
	InstanceName                  types.String `tfsdk:"instance_name"`
	IssueType                     types.String `tfsdk:"issue_type"`
	AlertTriggeredPayloadTemplate types.String `tfsdk:"alert_triggered_payload_template"`
	AlertResolvedPayloadTemplate  types.String `tfsdk:"alert_resolved_payload_template"`
}

var (
	_ resource.Resource                = (*ResourceServiceNow)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceServiceNow)(nil)
	_ resource.ResourceWithImportState = (*ResourceServiceNow)(nil)
)

func NewResourceServiceNow() resource.Resource { return &ResourceServiceNow{} }

func (serviceNow *ResourceServiceNow) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_now_integration"
}

func (serviceNow *ResourceServiceNow) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["username"] = schema.StringAttribute{
		Required:    true,
		Description: "User name used to authenticate the ServiceNow integration.",
	}
	attributes["password"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "Password used to authenticate the ServiceNow integration.",
	}
	attributes["instance_name"] = schema.StringAttribute{
		Required:    true,
		Description: "Name of the ServiceNow instance, for example `myinstance.service-now.com`.",
	}
	attributes["issue_type"] = schema.StringAttribute{
		Required:    true,
		Description: "Type of ServiceNow issue in standard ITIL terminology. Allowed values are `Incident` and `Problem`.",
		Validators: []validator.String{
			stringvalidator.OneOf(serviceNowTypeIncident, serviceNowTypeProblem),
		},
	}
	attributes["alert_triggered_payload_template"] = schema.StringAttribute{
		Optional:    true,
		Description: "Template used to create the ServiceNow POST JSON payload when an alert is triggered.",
	}
	attributes["alert_resolved_payload_template"] = schema.StringAttribute{
		Optional:    true,
		Description: "Template used to create the ServiceNow PUT JSON payload when an alert is resolved.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages a ServiceNow integration in Splunk Observability Cloud.",
		Attributes:  attributes,
	}
}

func (serviceNow *ResourceServiceNow) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceServiceNowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := serviceNow.Details().Client.CreateServiceNowIntegration(ctx, model.serviceNowIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (serviceNow *ResourceServiceNow) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceServiceNowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := serviceNow.Details().Client.GetServiceNowIntegration(ctx, model.ID.ValueString())
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

func (serviceNow *ResourceServiceNow) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceServiceNowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := serviceNow.Details().Client.UpdateServiceNowIntegration(
		ctx, model.ID.ValueString(), model.serviceNowIntegration(),
	)
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

func (serviceNow *ResourceServiceNow) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceServiceNowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, serviceNow.Details().Client.DeleteServiceNowIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceServiceNowModel) serviceNowIntegration() *integration.ServiceNowIntegration {
	return &integration.ServiceNowIntegration{
		Type:                          integration.SERVICE_NOW,
		Name:                          model.Name.ValueString(),
		Enabled:                       model.Enabled.ValueBool(),
		Username:                      model.Username.ValueString(),
		Password:                      model.Password.ValueString(),
		InstanceName:                  model.InstanceName.ValueString(),
		IssueType:                     model.IssueType.ValueString(),
		AlertTriggeredPayloadTemplate: model.AlertTriggeredPayloadTemplate.ValueString(),
		AlertResolvedPayloadTemplate:  model.AlertResolvedPayloadTemplate.ValueString(),
	}
}

func (model *resourceServiceNowModel) updateFromAPI(details *integration.ServiceNowIntegration, updateID bool) {
	if details == nil {
		return
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	model.InstanceName = types.StringValue(details.InstanceName)
	model.IssueType = types.StringValue(details.IssueType)
	updateOptionalString(&model.Username, details.Username)
	updateOptionalString(&model.Password, details.Password)
	updateOptionalString(&model.AlertTriggeredPayloadTemplate, details.AlertTriggeredPayloadTemplate)
	updateOptionalString(&model.AlertResolvedPayloadTemplate, details.AlertResolvedPayloadTemplate)
}
