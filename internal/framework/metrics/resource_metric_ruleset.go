// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metricruleset "github.com/signalfx/signalfx-go/metric_ruleset"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceMetricRuleset struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

var (
	_ resource.Resource                = (*ResourceMetricRuleset)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceMetricRuleset)(nil)
	_ resource.ResourceWithImportState = (*ResourceMetricRuleset)(nil)
)

func NewResourceMetricRuleset() resource.Resource {
	return &ResourceMetricRuleset{}
}

func (ruleset *ResourceMetricRuleset) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_ruleset"
}

func (ruleset *ResourceMetricRuleset) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a metric ruleset for Metrics Pipeline Management in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"metric_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the input metric controlled by the ruleset.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Optimistic-concurrency version of the metric ruleset.",
			},
			"description": optionalMetricRulesetString("Information about the metric ruleset."),
			"creator": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who created the metric ruleset.",
			},
			"created": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation timestamp in Unix milliseconds.",
			},
			"last_updated_by": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who most recently updated the metric ruleset.",
			},
			"last_updated": schema.Int64Attribute{
				Computed:    true,
				Description: "Most recent update timestamp in Unix milliseconds.",
			},
			"last_updated_by_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the user who most recently updated the metric ruleset.",
			},
		},
		Blocks: map[string]schema.Block{
			"aggregation_rules": schema.ListNestedBlock{
				Description: "Ordered aggregation rules for the metric.",
				NestedObject: schema.NestedBlockObject{
					Attributes: metricRulesetRuleAttributes("aggregation"),
					Blocks: map[string]schema.Block{
						"matcher":    metricRulesetMatcherBlock(),
						"aggregator": metricRulesetAggregatorBlock(),
					},
				},
			},
			"exception_rules": schema.ListNestedBlock{
				Description: "Ordered exception rules that reroute matching metric time series.",
				NestedObject: schema.NestedBlockObject{
					Attributes: metricRulesetRuleAttributes("exception"),
					Blocks: map[string]schema.Block{
						"matcher":     metricRulesetMatcherBlock(),
						"restoration": metricRulesetRestorationBlock(),
					},
				},
			},
			"routing_rule": schema.SingleNestedBlock{
				Description: "Required destination for the input metric.",
				Validators:  []validator.Object{objectvalidator.IsRequired()},
				Attributes: map[string]schema.Attribute{
					"destination": schema.StringAttribute{
						Required:    true,
						Description: "Routing destination: `RealTime`, `Archived`, or `Drop`.",
						Validators:  []validator.String{stringvalidator.OneOf("RealTime", "Archived", "Drop")},
					},
				},
			},
		},
	}
}

func optionalMetricRulesetString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(""),
		Description: description,
	}
}

func metricRulesetRuleAttributes(kind string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name":        optionalMetricRulesetString("Name of this " + kind + " rule."),
		"description": optionalMetricRulesetString("Information about this " + kind + " rule."),
		"enabled": schema.BoolAttribute{
			Required:    true,
			Description: "Whether this " + kind + " rule is active.",
		},
	}
}

func metricRulesetMatcherBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Required dimension matcher for this rule.",
		Validators:  []validator.Object{objectvalidator.IsRequired()},
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Matcher type. Must be `dimension`.",
				Validators:  []validator.String{stringvalidator.OneOf("dimension")},
			},
		},
		Blocks: map[string]schema.Block{
			"filters": schema.ListNestedBlock{
				Description: "Ordered dimension filters applied by the matcher.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"not": schema.BoolAttribute{
						Required:    true,
						Description: "Whether the filter excludes instead of includes matching values.",
					},
					"property": schema.StringAttribute{
						Required:    true,
						Description: "Dimension or custom property name to match.",
					},
					"property_value": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "Dimension or custom property values to match.",
					},
				}},
			},
		},
	}
}

func metricRulesetAggregatorBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Required rollup aggregator for this aggregation rule.",
		Validators:  []validator.Object{objectvalidator.IsRequired()},
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Aggregator type. Must be `rollup`.",
				Validators:  []validator.String{stringvalidator.OneOf("rollup")},
			},
			"output_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the aggregated output metric.",
			},
			"dimensions": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Dimensions to keep or drop in the aggregated metric.",
			},
			"drop_dimensions": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to drop the listed dimensions instead of retaining them.",
			},
		},
	}
}

func metricRulesetRestorationBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Optional restoration job that reroutes archived data to real time.",
		Attributes: map[string]schema.Attribute{
			"restoration_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the restoration job.",
			},
			"start_time": schema.Int64Attribute{
				Required:    true,
				Description: "Start of the restoration interval in Unix milliseconds.",
			},
			"stop_time": schema.Int64Attribute{
				Optional:    true,
				Description: "End of the restoration interval in Unix milliseconds.",
			},
		},
	}
}

func (ruleset *ResourceMetricRuleset) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceMetricRulesetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diagnostics := model.createRequest(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := ruleset.Details().Client.CreateMetricRuleset(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil || details.Id == nil || *details.Id == "" {
		resp.Diagnostics.AddError("Unable to create metric ruleset", "The metric ruleset API returned no resource identifier.")
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (ruleset *ResourceMetricRuleset) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceMetricRulesetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := ruleset.Details().Client.GetMetricRuleset(ctx, model.ID.ValueString())
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

func (ruleset *ResourceMetricRuleset) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceMetricRulesetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := ruleset.Details().Client.GetMetricRuleset(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if current == nil || current.Version == nil {
		resp.Diagnostics.AddError("Unable to update metric ruleset", "The metric ruleset API returned no current version.")
		return
	}
	payload, diagnostics := model.updateRequest(ctx, current.Version)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := ruleset.Details().Client.UpdateMetricRuleset(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.Diagnostics.AddError("Unable to update metric ruleset", "The metric ruleset API returned no updated ruleset.")
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (ruleset *ResourceMetricRuleset) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceMetricRulesetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, ruleset.Details().Client.DeleteMetricRuleset(ctx, model.ID.ValueString()))...)
	}
}

var _ metricRulesetResponse = (*metricruleset.CreateMetricRulesetResponse)(nil)
var _ metricRulesetResponse = (*metricruleset.GetMetricRulesetResponse)(nil)
var _ metricRulesetResponse = (*metricruleset.UpdateMetricRulesetResponse)(nil)
