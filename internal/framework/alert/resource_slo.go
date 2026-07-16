// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/slo"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceSLO struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

var (
	_ resource.Resource                   = (*ResourceSLO)(nil)
	_ resource.ResourceWithConfigure      = (*ResourceSLO)(nil)
	_ resource.ResourceWithImportState    = (*ResourceSLO)(nil)
	_ resource.ResourceWithValidateConfig = (*ResourceSLO)(nil)
	_ resource.ResourceWithModifyPlan     = (*ResourceSLO)(nil)
)

func NewResourceSLO() resource.Resource {
	return &ResourceSLO{}
}

func (r *ResourceSLO) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slo"
}

func (r *ResourceSLO) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a service-level objective (SLO) in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"name": schema.StringAttribute{
				Required: true, Description: "Unique name of the SLO.",
				Validators: []validator.String{stringvalidator.LengthBetween(0, 256)},
			},
			"description": schema.StringAttribute{
				Optional: true, Computed: true, Description: "Description of the SLO.",
				Validators: []validator.String{stringvalidator.LengthAtMost(1024)},
			},
			"type": schema.StringAttribute{
				Required: true, Description: "SLO input type. Currently only RequestBased is supported.",
				Validators: []validator.String{stringvalidator.OneOf(slo.RequestBased)},
			},
		},
		Blocks: map[string]schema.Block{
			"input":  sloInputBlock(),
			"target": sloTargetBlock(),
		},
	}
}

func sloInputBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "SignalFlow program and labels defining successful and total event streams.",
		Validators:  []validator.List{listvalidator.SizeBetween(1, 1)},
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"program_text": schema.StringAttribute{
				Required: true, Description: "SignalFlow program text for the SLO.",
				Validators: []validator.String{stringvalidator.LengthBetween(18, 50000)},
			},
			"good_events_label":  sloOptionalNonEmptyString("Program label for successful events."),
			"total_events_label": sloOptionalNonEmptyString("Program label for total events."),
		}},
	}
}

func sloTargetBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Target value and compliance window for the SLO.",
		Validators:  []validator.List{listvalidator.SizeBetween(1, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required: true, Description: "Target window type.",
					Validators: []validator.String{stringvalidator.OneOf(slo.RollingWindowTarget, slo.CalendarWindowTarget)},
				},
				"slo": schema.Float64Attribute{
					Required: true, Description: "Target percentage.",
					Validators: []validator.Float64{float64validator.Between(0, 100)},
				},
				"compliance_period": sloOptionalNonEmptyString("Compliance period for a rolling-window target."),
				"cycle_type": schema.StringAttribute{
					Optional: true, Description: "Calendar cycle type.",
					Validators: []validator.String{stringvalidator.OneOf("week", "month")},
				},
				"cycle_start": schema.StringAttribute{
					Optional: true, Computed: true, Description: "Calendar cycle start returned by the API when omitted.",
					Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
				},
			},
			Blocks: map[string]schema.Block{"alert_rule": sloAlertRuleBlock()},
		},
	}
}

func sloAlertRuleBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Ordered SLO alert-rule groups.",
		Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required: true, Description: "SLO alert-rule type.",
					Validators: []validator.String{stringvalidator.OneOf(slo.BreachRule, slo.ErrorBudgetLeftRule, slo.BurnRateRule)},
				},
			},
			Blocks: map[string]schema.Block{"rule": sloRuleBlock()},
		},
	}
}

func sloRuleBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Ordered detector rules for this SLO alert type.",
		Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"severity": schema.StringAttribute{
					Required: true, Description: "Severity of the rule.",
					Validators: []validator.String{stringvalidator.OneOf("Critical", "Warning", "Major", "Minor", "Info")},
				},
				"description":           sloOptionalComputedString("Description of the rule."),
				"notifications":         sloNotificationListAttribute(),
				"disabled":              schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this rule is disabled."},
				"parameterized_body":    sloOptionalComputedString("Custom notification body."),
				"parameterized_subject": sloOptionalComputedString("Custom notification subject."),
				"runbook_url":           sloOptionalComputedString("Runbook URL."),
				"tip":                   sloOptionalComputedString("Suggested first course of action."),
				"skip_clear_notification_states": schema.SetAttribute{
					Optional: true, Computed: true, ElementType: types.StringType,
					Description: "Alert clear states that do not send clear notifications.",
					Validators: []validator.Set{setvalidator.ValueStringsAre(
						stringvalidator.OneOf("OK", "AUTO_RESOLVED", "STOPPED", "MANUALLY_RESOLVED"),
					)},
				},
			},
			Blocks: map[string]schema.Block{
				"parameters":            sloParametersBlock(),
				"reminder_notification": sloReminderBlock(),
			},
		},
	}
}

func sloParametersBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Type-specific SLO alert parameters. Omitted values use API defaults.",
		Validators:  []validator.List{listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"fire_lasting":              sloOptionalComputedString("Duration before a breach or error-budget alert fires."),
			"percent_of_lasting":        schema.Float64Attribute{Optional: true, Computed: true, Description: "Required percentage of the fire-lasting duration."},
			"percent_error_budget_left": schema.Float64Attribute{Optional: true, Computed: true, Description: "Remaining error-budget percentage threshold."},
			"short_window_1":            sloOptionalComputedString("First short burn-rate window."),
			"long_window_1":             sloOptionalComputedString("First long burn-rate window."),
			"short_window_2":            sloOptionalComputedString("Second short burn-rate window."),
			"long_window_2":             sloOptionalComputedString("Second long burn-rate window."),
			"burn_rate_threshold_1":     schema.Float64Attribute{Optional: true, Computed: true, Description: "First burn-rate threshold."},
			"burn_rate_threshold_2":     schema.Float64Attribute{Optional: true, Computed: true, Description: "Second burn-rate threshold."},
		}},
	}
}

func sloReminderBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Optional repeated-notification settings.",
		Validators:  []validator.List{listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"interval_ms": schema.Int64Attribute{
				Required: true, Description: "Notification interval in milliseconds.",
				Validators: []validator.Int64{int64validator.AtLeast(0)},
			},
			"timeout_ms": schema.Int64Attribute{
				Optional: true, Description: "Notification timeout in milliseconds.",
				Validators: []validator.Int64{int64validator.AtLeast(0)},
			},
			"type": schema.StringAttribute{
				Required: true, Description: "Reminder type. Must be TIMEOUT.",
				Validators: []validator.String{stringvalidator.OneOf("TIMEOUT")},
			},
		}},
	}
}

func sloOptionalNonEmptyString(description string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Description: description, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}}
}

func sloOptionalComputedString(description string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, Description: description}
}

func sloNotificationListAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		Optional: true, Computed: true, ElementType: types.StringType,
		Description: "Ordered comma-delimited notification destinations.",
		Validators:  []validator.List{fwshared.NotificationStringListValidator()},
	}
}

func (r *ResourceSLO) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model resourceSLOModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(model.validate(ctx)...)
	}
}

func (r *ResourceSLO) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.Details() == nil {
		return
	}
	var model resourceSLOModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.hasUnknownRequiredValues(ctx) {
		return
	}
	payload, diagnostics := model.request(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !req.State.Raw.IsNull() {
		var current resourceSLOModel
		resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if id := current.ID.ValueString(); id != "" {
			payload.Name += " terraform-validation-" + id
		}
	}
	if err := r.Details().Client.ValidateSlo(ctx, payload); err != nil {
		detail := err.Error()
		if responseError, ok := signalfx.AsResponseError(err); ok {
			detail = fmt.Sprintf("%s: %q", err, responseError.Details())
		}
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid SLO configuration", detail)
	}
}

func (r *ResourceSLO) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceSLOModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.request(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.CreateSlo(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceSLO) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceSLOModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.GetSlo(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceSLO) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceSLOModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.request(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.UpdateSlo(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceSLO) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceSLOModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteSlo(ctx, model.ID.ValueString()))...)
	}
}
