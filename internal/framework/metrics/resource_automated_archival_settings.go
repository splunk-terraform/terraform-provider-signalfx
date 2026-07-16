// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	automatedarchival "github.com/signalfx/signalfx-go/automated-archival"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceAutomatedArchivalSettings struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceAutomatedArchivalSettingsModel struct {
	ID             types.String `tfsdk:"id"`
	Creator        types.String `tfsdk:"creator"`
	LastUpdatedBy  types.String `tfsdk:"last_updated_by"`
	Created        types.Int64  `tfsdk:"created"`
	LastUpdated    types.Int64  `tfsdk:"last_updated"`
	Version        types.String `tfsdk:"version"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	LookbackPeriod types.String `tfsdk:"lookback_period"`
	GracePeriod    types.String `tfsdk:"grace_period"`
	RulesetLimit   types.Int32  `tfsdk:"ruleset_limit"`
}

var (
	_ resource.Resource                = (*ResourceAutomatedArchivalSettings)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceAutomatedArchivalSettings)(nil)
	_ resource.ResourceWithImportState = (*ResourceAutomatedArchivalSettings)(nil)
)

func NewResourceAutomatedArchivalSettings() resource.Resource {
	return &ResourceAutomatedArchivalSettings{}
}

func (settings *ResourceAutomatedArchivalSettings) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automated_archival_settings"
}

func (settings *ResourceAutomatedArchivalSettings) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages organization-wide automated metric archival settings in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who created the automated archival settings.",
			},
			"last_updated_by": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who most recently updated the automated archival settings.",
			},
			"created": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation timestamp returned by Splunk Observability Cloud.",
			},
			"last_updated": schema.Int64Attribute{
				Computed:    true,
				Description: "Most recent update timestamp returned by Splunk Observability Cloud.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Optimistic-concurrency version of the automated archival settings.",
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether automated metric archival is enabled for the organization.",
			},
			"lookback_period": schema.StringAttribute{
				Required:    true,
				Description: "Unused-metric lookback period in ISO 8601 duration format, such as P30D, P45D, or P60D.",
			},
			"grace_period": schema.StringAttribute{
				Required:    true,
				Description: "Protection period for newly created metrics in ISO 8601 duration format, such as P0D, P15D, P30D, P45D, or P60D.",
			},
			"ruleset_limit": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Organization limit for the number of metric rulesets that can be created.",
			},
		},
	}
}

func (settings *ResourceAutomatedArchivalSettings) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAutomatedArchivalSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.toAPI()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := settings.Details().Client.CreateSettings(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.Diagnostics.AddError("Unable to create automated archival settings", "The automated archival API returned no settings.")
		return
	}
	model.ID = types.StringValue(strconv.FormatInt(details.Version, 10))
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (settings *ResourceAutomatedArchivalSettings) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceAutomatedArchivalSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := settings.Details().Client.GetSettings(ctx)
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

func (settings *ResourceAutomatedArchivalSettings) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceAutomatedArchivalSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior resourceAutomatedArchivalSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.copyComputedFrom(prior)
	payload, diagnostics := model.toAPI()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := settings.Details().Client.UpdateSettings(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.Diagnostics.AddError("Unable to update automated archival settings", "The automated archival API returned no settings.")
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (settings *ResourceAutomatedArchivalSettings) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceAutomatedArchivalSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	version, err := model.latestVersion()
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete automated archival settings", err.Error())
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, settings.Details().Client.DeleteSettings(
		ctx, &automatedarchival.AutomatedArchivalSettingsDeleteRequest{Version: &version},
	))...)
}

func (model resourceAutomatedArchivalSettingsModel) toAPI() (*automatedarchival.AutomatedArchivalSettings, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	payload := &automatedarchival.AutomatedArchivalSettings{
		Enabled:        model.Enabled.ValueBool(),
		LookbackPeriod: model.LookbackPeriod.ValueString(),
		GracePeriod:    model.GracePeriod.ValueString(),
	}
	if !model.Version.IsNull() && !model.Version.IsUnknown() && model.Version.ValueString() != "" {
		version, err := strconv.ParseInt(model.Version.ValueString(), 10, 64)
		if err != nil {
			diagnostics.AddError("Invalid automated archival settings version", err.Error())
		} else {
			payload.Version = version
		}
	}
	if !model.Creator.IsNull() && !model.Creator.IsUnknown() {
		value := model.Creator.ValueString()
		payload.Creator = &value
	}
	if !model.LastUpdatedBy.IsNull() && !model.LastUpdatedBy.IsUnknown() {
		value := model.LastUpdatedBy.ValueString()
		payload.LastUpdatedBy = &value
	}
	if !model.Created.IsNull() && !model.Created.IsUnknown() {
		value := model.Created.ValueInt64()
		payload.Created = &value
	}
	if !model.LastUpdated.IsNull() && !model.LastUpdated.IsUnknown() {
		value := model.LastUpdated.ValueInt64()
		payload.LastUpdated = &value
	}
	if !model.RulesetLimit.IsNull() && !model.RulesetLimit.IsUnknown() {
		value := model.RulesetLimit.ValueInt32()
		payload.RulesetLimit = &value
	}
	return payload, diagnostics
}

func (model *resourceAutomatedArchivalSettingsModel) updateFromAPI(details *automatedarchival.AutomatedArchivalSettings, preservePlanned bool) {
	if details == nil {
		return
	}
	if !preservePlanned {
		model.Enabled = types.BoolValue(details.Enabled)
		model.LookbackPeriod = types.StringValue(details.LookbackPeriod)
		model.GracePeriod = types.StringValue(details.GracePeriod)
	}
	model.Version = types.StringValue(strconv.FormatInt(details.Version, 10))
	model.Creator = automatedArchivalOptionalStringValue(model.Creator, details.Creator)
	model.LastUpdatedBy = automatedArchivalOptionalStringValue(model.LastUpdatedBy, details.LastUpdatedBy)
	model.Created = automatedArchivalOptionalInt64Value(model.Created, details.Created)
	model.LastUpdated = automatedArchivalOptionalInt64Value(model.LastUpdated, details.LastUpdated)
	if details.RulesetLimit != nil && (!preservePlanned || model.RulesetLimit.IsNull() || model.RulesetLimit.IsUnknown()) {
		model.RulesetLimit = types.Int32Value(*details.RulesetLimit)
	} else if details.RulesetLimit == nil && model.RulesetLimit.IsUnknown() {
		model.RulesetLimit = types.Int32Null()
	}
}

func (model *resourceAutomatedArchivalSettingsModel) copyComputedFrom(prior resourceAutomatedArchivalSettingsModel) {
	model.ID = prior.ID
	model.Creator = prior.Creator
	model.LastUpdatedBy = prior.LastUpdatedBy
	model.Created = prior.Created
	model.LastUpdated = prior.LastUpdated
	model.Version = prior.Version
	if model.RulesetLimit.IsUnknown() {
		model.RulesetLimit = prior.RulesetLimit
	}
}

func (model resourceAutomatedArchivalSettingsModel) latestVersion() (int64, error) {
	value := model.Version.ValueString()
	if model.Version.IsNull() || model.Version.IsUnknown() || value == "" {
		value = model.ID.ValueString()
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("version %q is not a valid integer: %w", value, err)
	}
	return version, nil
}
