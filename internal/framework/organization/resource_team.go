// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/team"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

const teamAppPath = "/team"

type ResourceTeam struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceTeamModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Members               types.Set    `tfsdk:"members"`
	NotificationsCritical types.List   `tfsdk:"notifications_critical"`
	NotificationsDefault  types.List   `tfsdk:"notifications_default"`
	NotificationsInfo     types.List   `tfsdk:"notifications_info"`
	NotificationsMajor    types.List   `tfsdk:"notifications_major"`
	NotificationsMinor    types.List   `tfsdk:"notifications_minor"`
	NotificationsWarning  types.List   `tfsdk:"notifications_warning"`
	URL                   types.String `tfsdk:"url"`
}

var (
	_ resource.Resource                = (*ResourceTeam)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceTeam)(nil)
	_ resource.ResourceWithImportState = (*ResourceTeam)(nil)
)

func NewResourceTeam() resource.Resource {
	return &ResourceTeam{}
}

func (teamResource *ResourceTeam) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (teamResource *ResourceTeam) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	notificationValidators := []validator.List{fwshared.NotificationStringListValidator()}
	resp.Schema = schema.Schema{
		Description: "Manages a team and its alert notification policies in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the team.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the team.",
			},
			"members": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Organization member IDs included in the team.",
			},
			"notifications_critical": notificationListAttribute("critical", notificationValidators),
			"notifications_default":  notificationListAttribute("default", notificationValidators),
			"notifications_info":     notificationListAttribute("info", notificationValidators),
			"notifications_major":    notificationListAttribute("major", notificationValidators),
			"notifications_minor":    notificationListAttribute("minor", notificationValidators),
			"notifications_warning":  notificationListAttribute("warning", notificationValidators),
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Splunk Observability Cloud application URL for the team.",
			},
		},
	}
}

func notificationListAttribute(category string, validators []validator.List) schema.ListAttribute {
	return schema.ListAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: "Ordered notification destinations for " + category + " alerts, in comma-delimited provider format.",
		Validators:  validators,
	}
}

func (teamResource *ResourceTeam) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := teamResource.Details().Client.CreateTeam(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil || details.Id == "" {
		resp.Diagnostics.AddError("Unable to create team", "The team API returned no resource identifier.")
		return
	}

	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true, pmeta.LoadApplicationURL(ctx, teamResource.Details(), teamAppPath, details.Id))...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (teamResource *ResourceTeam) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := teamResource.Details().Client.GetTeam(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	id := model.ID.ValueString()
	if details.Id != "" {
		id = details.Id
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, false, pmeta.LoadApplicationURL(ctx, teamResource.Details(), teamAppPath, id))...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (teamResource *ResourceTeam) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := teamResource.Details().Client.UpdateTeam(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	id := model.ID.ValueString()
	if details.Id != "" {
		id = details.Id
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true, pmeta.LoadApplicationURL(ctx, teamResource.Details(), teamAppPath, id))...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (teamResource *ResourceTeam) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, teamResource.Details().Client.DeleteTeam(ctx, model.ID.ValueString()))...)
}

func (model resourceTeamModel) createUpdateRequest(ctx context.Context) (*team.CreateUpdateTeamRequest, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	payload := &team.CreateUpdateTeamRequest{
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
	}

	if !model.Members.IsNull() {
		diagnostics.Append(model.Members.ElementsAs(ctx, &payload.Members, false)...)
	}

	notificationValues := []struct {
		value       types.List
		destination *[]*notification.Notification
	}{
		{model.NotificationsDefault, &payload.NotificationLists.Default},
		{model.NotificationsInfo, &payload.NotificationLists.Info},
		{model.NotificationsMinor, &payload.NotificationLists.Minor},
		{model.NotificationsWarning, &payload.NotificationLists.Warning},
		{model.NotificationsMajor, &payload.NotificationLists.Major},
		{model.NotificationsCritical, &payload.NotificationLists.Critical},
	}
	for _, item := range notificationValues {
		values, valueDiagnostics := fwshared.NotificationStringsToAPI(ctx, item.value)
		diagnostics.Append(valueDiagnostics...)
		*item.destination = values
	}
	return payload, diagnostics
}

func (model *resourceTeamModel) updateFromAPI(ctx context.Context, details *team.Team, updateID bool, applicationURL string) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if details == nil {
		return diagnostics
	}
	if updateID && details.Id != "" {
		model.ID = types.StringValue(details.Id)
	}
	model.Name = types.StringValue(details.Name)
	if details.Description != "" || !model.Description.IsNull() {
		model.Description = types.StringValue(details.Description)
	}
	if len(details.Members) > 0 || !model.Members.IsNull() {
		value, valueDiagnostics := types.SetValueFrom(ctx, types.StringType, details.Members)
		diagnostics.Append(valueDiagnostics...)
		model.Members = value
	}

	notificationValues := []struct {
		current     *types.List
		destination []*notification.Notification
	}{
		{&model.NotificationsDefault, details.NotificationLists.Default},
		{&model.NotificationsInfo, details.NotificationLists.Info},
		{&model.NotificationsMinor, details.NotificationLists.Minor},
		{&model.NotificationsWarning, details.NotificationLists.Warning},
		{&model.NotificationsMajor, details.NotificationLists.Major},
		{&model.NotificationsCritical, details.NotificationLists.Critical},
	}
	for _, item := range notificationValues {
		value, valueDiagnostics := fwshared.NotificationStringsFromAPI(ctx, *item.current, item.destination)
		diagnostics.Append(valueDiagnostics...)
		*item.current = value
	}
	model.URL = types.StringValue(applicationURL)
	return diagnostics
}
