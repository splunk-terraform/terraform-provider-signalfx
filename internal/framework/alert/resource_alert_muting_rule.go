// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/alertmuting"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

const alertMutingDetectorIDProperty = "sf_detectorId"

type ResourceAlertMutingRule struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type alertMutingRuleModel struct {
	ID                 types.String `tfsdk:"id"`
	Description        types.String `tfsdk:"description"`
	Detectors          types.List   `tfsdk:"detectors"`
	Filter             types.Set    `tfsdk:"filter"`
	Recurrence         types.Set    `tfsdk:"recurrence"`
	StartTime          types.Int64  `tfsdk:"start_time"`
	StopTime           types.Int64  `tfsdk:"stop_time"`
	EffectiveStartTime types.Int64  `tfsdk:"effective_start_time"`
}

type alertMutingRuleFilterModel struct {
	Property      types.String `tfsdk:"property"`
	PropertyValue types.String `tfsdk:"property_value"`
	Negated       types.Bool   `tfsdk:"negated"`
}

type alertMutingRuleRecurrenceModel struct {
	Unit  types.String `tfsdk:"unit"`
	Value types.Int64  `tfsdk:"value"`
}

var (
	_ resource.Resource                = &ResourceAlertMutingRule{}
	_ resource.ResourceWithConfigure   = &ResourceAlertMutingRule{}
	_ resource.ResourceWithImportState = &ResourceAlertMutingRule{}
)

func NewResourceAlertMutingRule() resource.Resource {
	return &ResourceAlertMutingRule{}
}

func (amr *ResourceAlertMutingRule) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_muting_rule"
}

func (amr *ResourceAlertMutingRule) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an alert muting rule.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"description": schema.StringAttribute{
				Required:    true,
				Description: "description of the rule",
			},
			"detectors": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "detectors to which this muting rule applies",
				Validators: []validator.List{
					listvalidator.AtLeastOneOf(path.MatchRoot("filter")),
				},
			},
			"start_time": schema.Int64Attribute{
				Required:    true,
				Description: "starting time of an alert muting rule as a Unix timestamp, in seconds",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"stop_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "stop time of an alert muting rule as a Unix timestamp, in seconds",
			},
			"effective_start_time": schema.Int64Attribute{
				Computed:    true,
				Description: "effective API start time in milliseconds",
			},
		},
		// TODO(v10): Once the v10 provider is served using protocol 6, migrate these
		// protocol-5-compatible blocks to schema.SetNestedAttribute so filter and
		// recurrence use the expected nested object attribute types.
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				Description: "list of alert muting filters for this rule",
				Validators: []validator.Set{
					setvalidator.AtLeastOneOf(path.MatchRoot("detectors")),
				},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"property": schema.StringAttribute{
						Required:    true,
						Description: "the property to filter by",
						Validators: []validator.String{
							stringvalidator.NoneOf(alertMutingDetectorIDProperty),
						},
					},
					"property_value": schema.StringAttribute{
						Required:    true,
						Description: "the value of the property to filter by",
					},
					"negated": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "(false by default) whether this filter should be a not filter",
					},
				}},
			},
			"recurrence": schema.SetNestedBlock{
				Description: "recurrence period for the muting rule",
				Validators: []validator.Set{
					setvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"unit": schema.StringAttribute{
						Required:    true,
						Description: "unit of the period. Can be days (d) or weeks (w)",
						Validators: []validator.String{
							stringvalidator.OneOf("d", "w"),
						},
					},
					"value": schema.Int64Attribute{
						Required:    true,
						Description: "amount of time, expressed as an integer applicable to the unit",
						Validators: []validator.Int64{
							int64validator.Between(1, math.MaxInt32),
						},
					},
				}},
			},
		},
	}
}

func (amr *ResourceAlertMutingRule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model alertMutingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := model.toRequest(ctx, false, time.Now())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := amr.Details().Client.CreateAlertMutingRule(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(model.updateFromRule(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (amr *ResourceAlertMutingRule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model alertMutingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := amr.Details().Client.GetAlertMutingRule(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}

	resp.Diagnostics.Append(model.updateFromRule(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (amr *ResourceAlertMutingRule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model alertMutingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior alertMutingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = prior.ID
	model.EffectiveStartTime = prior.EffectiveStartTime

	payload, diags := model.toRequest(ctx, true, time.Now())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := amr.Details().Client.UpdateAlertMutingRule(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(model.updateFromRule(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (amr *ResourceAlertMutingRule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model alertMutingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := amr.Details().Client.DeleteAlertMutingRule(ctx, model.ID.ValueString())
	if err != nil && strings.Contains(err.Error(), "400") {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
}

func (model alertMutingRuleModel) toRequest(ctx context.Context, update bool, now time.Time) (*alertmuting.CreateUpdateAlertMutingRuleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	payload := &alertmuting.CreateUpdateAlertMutingRuleRequest{
		Description: model.Description.ValueString(),
		StartTime:   model.StartTime.ValueInt64() * 1000,
		StopTime:    model.StopTime.ValueInt64() * 1000,
	}

	var filters []alertMutingRuleFilterModel
	if !model.Filter.IsNull() && !model.Filter.IsUnknown() {
		diags.Append(model.Filter.ElementsAs(ctx, &filters, false)...)
		for _, filter := range filters {
			payload.Filters = append(payload.Filters, &alertmuting.AlertMutingRuleFilter{
				Property: filter.Property.ValueString(),
				PropertyValue: alertmuting.StringOrArray{
					Values: []string{filter.PropertyValue.ValueString()},
				},
				NOT: filter.Negated.ValueBool(),
			})
		}
	}

	var detectors []string
	if !model.Detectors.IsNull() && !model.Detectors.IsUnknown() {
		diags.Append(model.Detectors.ElementsAs(ctx, &detectors, false)...)
		if len(detectors) > 0 {
			payload.Filters = append(payload.Filters, &alertmuting.AlertMutingRuleFilter{
				Property: alertMutingDetectorIDProperty,
				PropertyValue: alertmuting.StringOrArray{
					Values: detectors,
				},
			})
		}
	}

	var recurrence []alertMutingRuleRecurrenceModel
	if !model.Recurrence.IsNull() && !model.Recurrence.IsUnknown() {
		diags.Append(model.Recurrence.ElementsAs(ctx, &recurrence, false)...)
		if len(recurrence) > 0 {
			recurrenceValue := recurrence[0].Value.ValueInt64()
			payload.Recurrence = &alertmuting.AlertMutingRuleRecurrence{
				Unit:  recurrence[0].Unit.ValueString(),
				Value: int32(recurrenceValue), // #nosec G115 -- schema validation bounds this value to int32.
			}
		}
	}

	if update && model.StartTime.ValueInt64() <= now.Unix() && !model.EffectiveStartTime.IsNull() && !model.EffectiveStartTime.IsUnknown() {
		payload.StartTime = model.EffectiveStartTime.ValueInt64()
	}

	return payload, diags
}

func (model *alertMutingRuleModel) updateFromRule(ctx context.Context, details *alertmuting.AlertMutingRule) diag.Diagnostics {
	var diags diag.Diagnostics
	if details == nil {
		diags.AddError("Invalid alert muting rule response", "The alert muting rule API returned no resource data.")
		return diags
	}
	if details.Id == "" {
		diags.AddError("Invalid alert muting rule response", "The alert muting rule API returned no resource identifier.")
		return diags
	}
	model.ID = types.StringValue(details.Id)
	model.Description = types.StringValue(details.Description)
	model.EffectiveStartTime = types.Int64Value(details.StartTime)
	model.StopTime = types.Int64Value(details.StopTime / 1000)
	if model.StartTime.IsNull() || model.StartTime.IsUnknown() {
		model.StartTime = types.Int64Value(details.StartTime / 1000)
	}

	filters := make([]alertMutingRuleFilterModel, 0, len(details.Filters))
	detectors := make([]string, 0, len(details.Filters))
	for index, filter := range details.Filters {
		if filter == nil {
			diags.AddError("Invalid alert muting rule response", fmt.Sprintf("The alert muting rule API returned an empty filter at index %d.", index))
			continue
		}
		if filter.Property == alertMutingDetectorIDProperty {
			detectors = append(detectors, filter.PropertyValue.Values...)
			continue
		}

		value := ""
		switch len(filter.PropertyValue.Values) {
		case 0:
		case 1:
			value = filter.PropertyValue.Values[0]
		default:
			diags.AddError(
				"Unsupported alert muting rule filter",
				"terraform provider does not support arrays in alert muting rule filter values for property \""+filter.Property+"\"",
			)
			continue
		}

		filters = append(filters, alertMutingRuleFilterModel{
			Property:      types.StringValue(filter.Property),
			PropertyValue: types.StringValue(value),
			Negated:       types.BoolValue(filter.NOT),
		})
	}

	if len(detectors) == 0 {
		model.Detectors = types.ListNull(types.StringType)
	} else {
		model.Detectors, _ = types.ListValueFrom(ctx, types.StringType, detectors)
	}
	filterType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"property": types.StringType, "property_value": types.StringType, "negated": types.BoolType,
	}}
	if len(filters) == 0 {
		model.Filter = types.SetNull(filterType)
	} else {
		var setDiags diag.Diagnostics
		model.Filter, setDiags = types.SetValueFrom(ctx, filterType, filters)
		diags.Append(setDiags...)
	}

	recurrenceType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"unit": types.StringType, "value": types.Int64Type,
	}}
	if details.Recurrence == nil {
		model.Recurrence = types.SetNull(recurrenceType)
	} else {
		var setDiags diag.Diagnostics
		model.Recurrence, setDiags = types.SetValueFrom(ctx, recurrenceType, []alertMutingRuleRecurrenceModel{{
			Unit: types.StringValue(details.Recurrence.Unit), Value: types.Int64Value(int64(details.Recurrence.Value)),
		}})
		diags.Append(setDiags...)
	}

	return diags
}
