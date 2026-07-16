// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/detector"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/visual"
)

const detectorAppPath = "/detector/v2"

type ResourceDetector struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

var (
	_ resource.Resource                   = (*ResourceDetector)(nil)
	_ resource.ResourceWithConfigure      = (*ResourceDetector)(nil)
	_ resource.ResourceWithImportState    = (*ResourceDetector)(nil)
	_ resource.ResourceWithValidateConfig = (*ResourceDetector)(nil)
)

func NewResourceDetector() resource.Resource {
	return &ResourceDetector{}
}

func (r *ResourceDetector) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_detector"
}

func (r *ResourceDetector) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a detector and its alert rules in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the detector.",
			},
			"program_text": schema.StringAttribute{
				Required:    true,
				Description: "SignalFlow program text for the detector.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 50000)},
			},
			"description": detectorOptionalString("Description of the detector."),
			"timezone": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UTC"),
				Description: "Geographic time zone associated with the detector, for example `Australia/Sydney`.",
				Validators:  []validator.String{detectorTimeZoneValidator{}},
			},
			"max_delay": detectorDelayAttribute("Maximum time in seconds to wait for late datapoints."),
			"min_delay": detectorDelayAttribute("Minimum time in seconds to wait even when datapoints arrive on time."),
			"show_data_markers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to draw markers for datapoints in the visualization.",
			},
			"show_event_lines": detectorOptionalBool(false, "Whether to draw a vertical line for each triggered event."),
			"disable_sampling": detectorOptionalBool(false, "Whether to display all datapoints instead of sampling them."),
			"time_range": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Relative visualization range in seconds. Defaults to 3600 when absolute times are not configured.",
				Validators: []validator.Int64{
					int64validator.Between(0, detectorMaximumSeconds),
					int64validator.ConflictsWith(path.MatchRoot("start_time"), path.MatchRoot("end_time")),
				},
			},
			"start_time":              detectorAbsoluteTimeAttribute("Start of the absolute visualization range in Unix seconds."),
			"end_time":                detectorAbsoluteTimeAttribute("End of the absolute visualization range in Unix seconds."),
			"tags":                    detectorOptionalStringSet("Tags associated with the detector."),
			"teams":                   detectorOptionalStringSet("Team IDs associated with the detector."),
			"authorized_writer_teams": detectorOptionalStringSet("Team IDs with write access to the detector."),
			"authorized_writer_users": detectorOptionalStringSet("User IDs with write access to the detector."),
			"label_resolutions": schema.MapAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "Resolution in milliseconds for each published detector label.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "URL of the detector in Splunk Observability Cloud.",
			},
			"detector_origin": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Standard"),
				Description: "How the detector was created: `Standard` or `AutoDetectCustomization`. Changes replace the resource.",
				Validators:  []validator.String{stringvalidator.OneOf("Standard", "AutoDetectCustomization")},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_detector_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Parent AutoDetect detector ID for an AutoDetect customization. Changes replace the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rule":        detectorRuleBlock(),
			"viz_options": detectorVisualizationBlock(),
		},
	}
}

func detectorOptionalString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: description,
	}
}

func detectorOptionalBool(value bool, description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true, Computed: true, Default: booldefault.StaticBool(value), Description: description,
	}
}

func detectorNestedOptionalString(description string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Description: description}
}

func detectorDelayAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional: true, Computed: true, Default: int64default.StaticInt64(0), Description: description,
		Validators: []validator.Int64{int64validator.Between(0, 900)},
	}
}

func detectorAbsoluteTimeAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Validators: []validator.Int64{
			int64validator.Between(0, detectorMaximumSeconds),
			int64validator.ConflictsWith(path.MatchRoot("time_range")),
		},
	}
}

func detectorOptionalStringSet(description string) schema.SetAttribute {
	return schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: description}
}

func detectorRuleBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "Required set of alert rules.",
		Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"severity": schema.StringAttribute{
					Required:    true,
					Description: "Severity of the rule.",
					Validators:  []validator.String{stringvalidator.OneOf("Critical", "Warning", "Major", "Minor", "Info")},
				},
				"detect_label": schema.StringAttribute{Required: true, Description: "Publish label associated with this alert rule."},
				"description":  detectorNestedOptionalString("Description of the rule."),
				"notifications": schema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Ordered comma-delimited notification destinations.",
					Validators:  []validator.List{fwshared.NotificationStringListValidator()},
				},
				"disabled":              schema.BoolAttribute{Optional: true, Description: "Whether this alert rule is disabled."},
				"parameterized_body":    detectorNestedOptionalString("Custom notification body."),
				"parameterized_subject": detectorNestedOptionalString("Custom notification subject."),
				"runbook_url":           detectorNestedOptionalString("Runbook URL for the alert rule."),
				"tip":                   detectorNestedOptionalString("Suggested first course of action."),
				"skip_clear_notification_states": schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Alert clear states that do not send clear notifications.",
					Validators: []validator.Set{setvalidator.ValueStringsAre(
						stringvalidator.OneOf("OK", "AUTO_RESOLVED", "STOPPED", "MANUALLY_RESOLVED"),
					)},
				},
			},
			Blocks: map[string]schema.Block{
				"reminder_notification": schema.ListNestedBlock{
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
							Required: true, Description: "Reminder type. Must be `TIMEOUT`.",
							Validators: []validator.String{stringvalidator.OneOf("TIMEOUT")},
						},
					}},
				},
			},
		},
	}
}

func detectorVisualizationBlock() schema.SetNestedBlock {
	colors := append([]string{""}, visual.NewColorPalette().Names()...)
	return schema.SetNestedBlock{
		Description: "Per-publish-label visualization options.",
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"label":        schema.StringAttribute{Required: true, Description: "SignalFlow publish label."},
			"color":        detectorValidatedOptionalString("Color name.", stringvalidator.OneOf(colors...)),
			"display_name": detectorNestedOptionalString("Alternate display name."),
			"value_unit": detectorValidatedOptionalString(
				"Unit attached to the plot values.", stringvalidator.OneOf(append([]string{""}, detectorValueUnits...)...),
			),
			"value_prefix": detectorNestedOptionalString("Prefix displayed with plot values."),
			"value_suffix": detectorNestedOptionalString("Suffix displayed with plot values."),
		}},
	}
}

func detectorValidatedOptionalString(description string, validation validator.String) schema.StringAttribute {
	value := detectorNestedOptionalString(description)
	value.Validators = []validator.String{validation}
	return value
}

func (r *ResourceDetector) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model resourceDetectorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model.DetectorOrigin.ValueString() == "AutoDetectCustomization" && model.ParentDetectorID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(path.Root("parent_detector_id"), "Missing parent detector", "AutoDetectCustomization detectors require parent_detector_id.")
	}
	if model.Rules.IsNull() || (!model.Rules.IsUnknown() && len(model.Rules.Elements()) == 0) {
		resp.Diagnostics.AddAttributeError(path.Root("rule"), "Missing detector rule", "At least one detector rule is required.")
	}
	if !model.EndTime.IsNull() && !model.EndTime.IsUnknown() && model.StartTime.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("end_time"), "Missing detector start time", "An absolute detector end_time requires start_time.")
	}
	if r.Details() == nil || model.Name.IsUnknown() || model.ProgramText.IsUnknown() || model.Rules.IsUnknown() || resp.Diagnostics.HasError() {
		return
	}
	request, diagnostics := model.validationRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.Details().Client.ValidateDetector(ctx, request); err != nil {
		detail := err.Error()
		if responseError, ok := signalfx.AsResponseError(err); ok {
			detail = fmt.Sprintf("%s: %q", err, responseError.Details())
		}
		resp.Diagnostics.AddAttributeError(path.Root("program_text"), "Invalid detector program or rules", detail)
	}
}

func (r *ResourceDetector) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceDetectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.request(ctx, r.Details())
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.CreateDetector(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, detectorURL(ctx, r.Details(), details))...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceDetector) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceDetectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.GetDetector(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, detectorURL(ctx, r.Details(), details))...)
	if details.OverMTSLimit {
		resp.Diagnostics.AddWarning("Detector is over the MTS limit", "One or more detector statements matched too many metric time series and may evaluate incomplete data.")
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceDetector) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceDetectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.request(ctx, r.Details())
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := r.Details().Client.UpdateDetector(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, detectorURL(ctx, r.Details(), details))...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *ResourceDetector) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceDetectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteDetector(ctx, model.ID.ValueString()))...)
	}
}

func detectorURL(ctx context.Context, meta any, details *detector.Detector) string {
	if details == nil || details.Id == "" {
		return ""
	}
	return pmeta.LoadApplicationURL(ctx, meta, detectorAppPath, details.Id, "edit")
}

var detectorValueUnits = []string{
	"Bit", "Kilobit", "Megabit", "Gigabit", "Terabit", "Petabit", "Exabit", "Zettabit", "Yottabit",
	"Byte", "Kibibyte", "Mebibyte", "Gibibyte", "Tebibyte", "Pebibyte", "Exbibyte", "Zebibyte", "Yobibyte",
	"Nanosecond", "Microsecond", "Millisecond", "Second", "Minute", "Hour", "Day", "Week",
}

const (
	detectorMaximumDelayMilliseconds = int64(math.MaxInt32)
	detectorMaximumSeconds           = int64(math.MaxInt64 / 1000)
)
