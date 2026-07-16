// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/orgtoken"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceOrgToken struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceOrgTokenModel struct {
	ID                types.String                    `tfsdk:"id"`
	Name              types.String                    `tfsdk:"name"`
	Description       types.String                    `tfsdk:"description"`
	AuthScopes        types.List                      `tfsdk:"auth_scopes"`
	Disabled          types.Bool                      `tfsdk:"disabled"`
	HostOrUsageLimits *orgTokenHostOrUsageLimitsModel `tfsdk:"host_or_usage_limits"`
	DPMLimits         *orgTokenDPMLimitsModel         `tfsdk:"dpm_limits"`
	Notifications     types.List                      `tfsdk:"notifications"`
	Secret            types.String                    `tfsdk:"secret"`
	ExpiresAt         types.Int64                     `tfsdk:"expires_at"`
}

type orgTokenHostOrUsageLimitsModel struct {
	HostNotificationThreshold           types.Int64 `tfsdk:"host_notification_threshold"`
	HostLimit                           types.Int64 `tfsdk:"host_limit"`
	ContainerNotificationThreshold      types.Int64 `tfsdk:"container_notification_threshold"`
	ContainerLimit                      types.Int64 `tfsdk:"container_limit"`
	CustomMetricsNotificationThreshold  types.Int64 `tfsdk:"custom_metrics_notification_threshold"`
	CustomMetricsLimit                  types.Int64 `tfsdk:"custom_metrics_limit"`
	HighResMetricsNotificationThreshold types.Int64 `tfsdk:"high_res_metrics_notification_threshold"`
	HighResMetricsLimit                 types.Int64 `tfsdk:"high_res_metrics_limit"`
}

type orgTokenDPMLimitsModel struct {
	DPMNotificationThreshold types.Int32 `tfsdk:"dpm_notification_threshold"`
	DPMLimit                 types.Int32 `tfsdk:"dpm_limit"`
}

var (
	_ resource.Resource                   = (*ResourceOrgToken)(nil)
	_ resource.ResourceWithConfigure      = (*ResourceOrgToken)(nil)
	_ resource.ResourceWithImportState    = (*ResourceOrgToken)(nil)
	_ resource.ResourceWithValidateConfig = (*ResourceOrgToken)(nil)
)

func NewResourceOrgToken() resource.Resource {
	return &ResourceOrgToken{}
}

func (tokenResource *ResourceOrgToken) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_token"
}

func (tokenResource *ResourceOrgToken) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an organization access token in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Organization token identifier returned by Splunk Observability Cloud.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the token.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the token.",
			},
			"auth_scopes": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ordered authentication scopes for the token, such as INGEST, API, or RUM.",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the token is disabled and unavailable for authentication.",
			},
			"notifications": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Ordered notification destinations for token limit alerts, in comma-delimited provider format.",
				Validators:  []validator.List{fwshared.NotificationStringListValidator()},
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Secret value assigned to the token by Splunk Observability Cloud.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Token expiration time as Unix milliseconds.",
			},
		},
		Blocks: map[string]schema.Block{
			"host_or_usage_limits": schema.SingleNestedBlock{
				Description: "Host-based or usage-based limits and notification thresholds for the token.",
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRoot("dpm_limits")),
				},
				Attributes: hostOrUsageLimitAttributes(),
			},
			"dpm_limits": schema.SingleNestedBlock{
				Description: "Datapoints-per-minute limit and notification threshold for the token.",
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRoot("host_or_usage_limits")),
				},
				Attributes: map[string]schema.Attribute{
					"dpm_limit": schema.Int32Attribute{
						Optional:    true,
						Description: "Maximum datapoints per minute accepted for the token. Required when the dpm_limits block is configured.",
					},
					"dpm_notification_threshold": schema.Int32Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(-1),
						Description: "DPM level at which Splunk Observability Cloud sends a notification; -1 disables the threshold.",
					},
				},
			},
		},
	}
}

func (tokenResource *ResourceOrgToken) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model resourceOrgTokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.DPMLimits == nil {
		return
	}
	if model.DPMLimits.DPMLimit.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("dpm_limits").AtName("dpm_limit"),
			"Missing DPM limit",
			"The dpm_limit attribute is required when the dpm_limits block is configured.",
		)
	}
}

func hostOrUsageLimitAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host_notification_threshold":           limitInt64Attribute("Notification threshold for hosts; -1 disables the threshold."),
		"host_limit":                            limitInt64Attribute("Maximum number of hosts that can use the token; -1 disables the limit."),
		"container_notification_threshold":      limitInt64Attribute("Notification threshold for containers; -1 disables the threshold."),
		"container_limit":                       limitInt64Attribute("Maximum number of containers that can use the token; -1 disables the limit."),
		"custom_metrics_notification_threshold": limitInt64Attribute("Notification threshold for custom metrics; -1 disables the threshold."),
		"custom_metrics_limit":                  limitInt64Attribute("Maximum number of custom metrics accepted for the token; -1 disables the limit."),
		"high_res_metrics_notification_threshold": limitInt64Attribute(
			"Notification threshold for high-resolution metrics; -1 disables the threshold.",
		),
		"high_res_metrics_limit": limitInt64Attribute(
			"Maximum number of high-resolution metrics accepted for the token; -1 disables the limit.",
		),
	}
}

func limitInt64Attribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(-1),
		Description: description,
	}
}

func (tokenResource *ResourceOrgToken) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceOrgTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := tokenResource.Details().Client.CreateOrgToken(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil || details.Name == "" {
		resp.Diagnostics.AddError("Unable to create organization token", "The organization token API returned no resource identifier.")
		return
	}

	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (tokenResource *ResourceOrgToken) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceOrgTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := tokenResource.Details().Client.GetOrgToken(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, false, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (tokenResource *ResourceOrgToken) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceOrgTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior resourceOrgTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := tokenResource.Details().Client.UpdateOrgToken(ctx, prior.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil || details.Name == "" {
		resp.Diagnostics.AddError("Unable to update organization token", "The organization token API returned no resource identifier.")
		return
	}

	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (tokenResource *ResourceOrgToken) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceOrgTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, tokenResource.Details().Client.DeleteOrgToken(ctx, model.ID.ValueString()))...)
}

func (model resourceOrgTokenModel) createUpdateRequest(ctx context.Context) (*orgtoken.CreateUpdateTokenRequest, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	payload := &orgtoken.CreateUpdateTokenRequest{
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
		Disabled:    model.Disabled.ValueBool(),
	}

	if !model.AuthScopes.IsNull() && !model.AuthScopes.IsUnknown() {
		diagnostics.Append(model.AuthScopes.ElementsAs(ctx, &payload.AuthScopes, false)...)
	}

	values, valueDiagnostics := fwshared.NotificationStringsToAPI(ctx, model.Notifications)
	diagnostics.Append(valueDiagnostics...)
	payload.Notifications = values

	switch {
	case model.HostOrUsageLimits != nil:
		payload.Limits = model.HostOrUsageLimits.toAPI(&diagnostics)
	case model.DPMLimits != nil:
		payload.Limits = model.DPMLimits.toAPI(&diagnostics)
	}

	return payload, diagnostics
}

func (model orgTokenHostOrUsageLimitsModel) toAPI(diagnostics *diag.Diagnostics) *orgtoken.Limit {
	quota := &orgtoken.UsageLimits{}
	threshold := &orgtoken.UsageLimits{}
	for _, item := range []struct {
		value       types.Int64
		destination **int64
		name        string
	}{
		{model.HostLimit, &quota.HostThreshold, "host_limit"},
		{model.ContainerLimit, &quota.ContainerThreshold, "container_limit"},
		{model.CustomMetricsLimit, &quota.CustomMetricThreshold, "custom_metrics_limit"},
		{model.HighResMetricsLimit, &quota.HighResMetricThreshold, "high_res_metrics_limit"},
		{model.HostNotificationThreshold, &threshold.HostThreshold, "host_notification_threshold"},
		{model.ContainerNotificationThreshold, &threshold.ContainerThreshold, "container_notification_threshold"},
		{model.CustomMetricsNotificationThreshold, &threshold.CustomMetricThreshold, "custom_metrics_notification_threshold"},
		{model.HighResMetricsNotificationThreshold, &threshold.HighResMetricThreshold, "high_res_metrics_notification_threshold"},
	} {
		if item.value.IsUnknown() {
			diagnostics.AddError("Unknown organization token limit", "The "+item.name+" value must be known before applying the resource.")
			continue
		}
		if value := item.value.ValueInt64(); !item.value.IsNull() && value != -1 {
			*item.destination = &value
		}
	}
	return &orgtoken.Limit{CategoryQuota: quota, CategoryNotificationThreshold: threshold}
}

func (model orgTokenDPMLimitsModel) toAPI(diagnostics *diag.Diagnostics) *orgtoken.Limit {
	if model.DPMLimit.IsUnknown() {
		diagnostics.AddError("Unknown organization token limit", "The dpm_limit value must be known before applying the resource.")
	}
	quota := model.DPMLimit.ValueInt32()
	limits := &orgtoken.Limit{DpmQuota: &quota}
	if model.DPMNotificationThreshold.IsUnknown() {
		diagnostics.AddError("Unknown organization token limit", "The dpm_notification_threshold value must be known before applying the resource.")
	} else if value := model.DPMNotificationThreshold.ValueInt32(); !model.DPMNotificationThreshold.IsNull() && value != -1 {
		limits.DpmNotificationThreshold = &value
	}
	return limits
}

func (model *resourceOrgTokenModel) updateFromAPI(
	ctx context.Context,
	details *orgtoken.Token,
	updateID bool,
	preservePlanned bool,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if details == nil {
		return diagnostics
	}
	if updateID && details.Name != "" {
		model.ID = types.StringValue(details.Name)
	}
	if model.Name.IsNull() || model.Name.IsUnknown() {
		model.Name = types.StringValue(details.Name)
	}
	if !preservePlanned && (details.Description != "" || !model.Description.IsNull()) {
		model.Description = types.StringValue(details.Description)
	}
	if !preservePlanned || model.Disabled.IsNull() || model.Disabled.IsUnknown() {
		model.Disabled = types.BoolValue(details.Disabled)
	}
	if !preservePlanned || model.AuthScopes.IsNull() || model.AuthScopes.IsUnknown() {
		scopes := append([]string(nil), details.AuthScopes...)
		sort.Strings(scopes)
		if len(scopes) > 0 || !model.AuthScopes.IsNull() {
			value, valueDiagnostics := types.ListValueFrom(ctx, types.StringType, scopes)
			diagnostics.Append(valueDiagnostics...)
			model.AuthScopes = value
		}
	}
	if !preservePlanned || model.Notifications.IsNull() || model.Notifications.IsUnknown() {
		value, valueDiagnostics := fwshared.NotificationStringsFromAPI(ctx, model.Notifications, details.Notifications)
		diagnostics.Append(valueDiagnostics...)
		model.Notifications = value
	}
	if !preservePlanned || (model.HostOrUsageLimits == nil && model.DPMLimits == nil) {
		model.updateLimitsFromAPI(details.Limits)
	}
	if details.Secret != "" {
		model.Secret = types.StringValue(details.Secret)
	} else if model.Secret.IsUnknown() {
		model.Secret = types.StringNull()
	}
	model.ExpiresAt = types.Int64Value(details.Expiry)
	return diagnostics
}

func (model *resourceOrgTokenModel) updateLimitsFromAPI(limits *orgtoken.Limit) {
	model.HostOrUsageLimits = nil
	model.DPMLimits = nil
	if limits == nil {
		return
	}
	if limits.DpmQuota != nil {
		model.DPMLimits = &orgTokenDPMLimitsModel{
			DPMLimit:                 types.Int32Value(*limits.DpmQuota),
			DPMNotificationThreshold: types.Int32Value(int32ValueOrDefault(limits.DpmNotificationThreshold, -1)),
		}
		return
	}
	if !usageLimitsPresent(limits.CategoryQuota) && !usageLimitsPresent(limits.CategoryNotificationThreshold) {
		return
	}
	model.HostOrUsageLimits = &orgTokenHostOrUsageLimitsModel{
		HostLimit:                           types.Int64Value(usageLimitValue(limits.CategoryQuota, func(value *orgtoken.UsageLimits) *int64 { return value.HostThreshold })),
		ContainerLimit:                      types.Int64Value(usageLimitValue(limits.CategoryQuota, func(value *orgtoken.UsageLimits) *int64 { return value.ContainerThreshold })),
		CustomMetricsLimit:                  types.Int64Value(usageLimitValue(limits.CategoryQuota, func(value *orgtoken.UsageLimits) *int64 { return value.CustomMetricThreshold })),
		HighResMetricsLimit:                 types.Int64Value(usageLimitValue(limits.CategoryQuota, func(value *orgtoken.UsageLimits) *int64 { return value.HighResMetricThreshold })),
		HostNotificationThreshold:           types.Int64Value(usageLimitValue(limits.CategoryNotificationThreshold, func(value *orgtoken.UsageLimits) *int64 { return value.HostThreshold })),
		ContainerNotificationThreshold:      types.Int64Value(usageLimitValue(limits.CategoryNotificationThreshold, func(value *orgtoken.UsageLimits) *int64 { return value.ContainerThreshold })),
		CustomMetricsNotificationThreshold:  types.Int64Value(usageLimitValue(limits.CategoryNotificationThreshold, func(value *orgtoken.UsageLimits) *int64 { return value.CustomMetricThreshold })),
		HighResMetricsNotificationThreshold: types.Int64Value(usageLimitValue(limits.CategoryNotificationThreshold, func(value *orgtoken.UsageLimits) *int64 { return value.HighResMetricThreshold })),
	}
}

func usageLimitsPresent(value *orgtoken.UsageLimits) bool {
	return value != nil && (value.HostThreshold != nil || value.ContainerThreshold != nil ||
		value.CustomMetricThreshold != nil || value.HighResMetricThreshold != nil)
}

func usageLimitValue(value *orgtoken.UsageLimits, field func(*orgtoken.UsageLimits) *int64) int64 {
	if value == nil || field(value) == nil {
		return -1
	}
	return *field(value)
}

func int32ValueOrDefault(value *int32, fallback int32) int32 {
	if value == nil {
		return fallback
	}
	return *value
}
