// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwchart

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	fwtypes "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/types"
)

type ResourceChart struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type ResourceChartModel struct {
	Id                   types.String `tfsdk:"id"`
	URL                  types.String `tfsdk:"url"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Tags                 types.List   `tfsdk:"tags"`
	ProgramOptions       types.Object `tfsdk:"program_options"`
	VisualizationOptions types.Object `tfsdk:"visualization_options"`
	DataOptions          types.Object `tfsdk:"data_options"`
	PublishOptions       types.List   `tfsdk:"publish_options"`
}

var (
	_ resource.Resource                = (*ResourceChart)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceChart)(nil)
	_ resource.ResourceWithImportState = (*ResourceChart)(nil)
)

func NewResourceChart() resource.Resource {
	return &ResourceChart{}
}

func (rc *ResourceChart) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart"
}

func (rc *ResourceChart) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Chart is used to display information with a dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The URL of the chart.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the chart.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the chart.",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The tags associated with the chart.",
				ElementType: types.StringType,
			},
			"program_options": schema.SingleNestedAttribute{
				Description: "Program Options is used to detail what data is being queried and additional options that can improve query performance.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"text": schema.StringAttribute{
						Required:    true,
						Description: "This is the program text used to query data stored in Splunk Observability Cloud.",
					},
					"min_resolution": schema.StringAttribute{
						Optional:   true,
						CustomType: timetypes.GoDurationType{},
					},
					"max_delay": schema.StringAttribute{
						Optional:   true,
						CustomType: timetypes.GoDurationType{},
					},
					"disable_sampling": schema.BoolAttribute{
						Optional: true,
					},
					"timezone": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("UTC"),
					},
				},
			},
			"visualization_options": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"heatmap": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"color_by": schema.StringAttribute{
								Optional: true,
								Validators: []validator.String{
									stringvalidator.OneOf("Range", "Scale"),
								},
							},
							"group_by": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
					"list": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"hide_missing_values": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Default:  booldefault.StaticBool(false),
							},
						},
					},
					"table": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"group_by": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"hide_missing_values": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Default:  booldefault.StaticBool(false),
							},
						},
					},
					"single_value": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"show_sparkline": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Default:  booldefault.StaticBool(false),
							},
						},
					},
					"text": schema.SingleNestedAttribute{
						Optional: true,
					},
					"time_series": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"color_by": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString("Dimension"),
								Validators: []validator.String{
									stringvalidator.OneOf("Metric", "Dimension"),
								},
							},
							"include_zero": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Default:  booldefault.StaticBool(false),
							},
							"show_data_markers": schema.BoolAttribute{
								Optional: true,
							},
							"show_legend": schema.BoolAttribute{
								Optional: true,
							},
							"show_event_lines": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Default:  booldefault.StaticBool(true),
							},
							"sort_by": schema.StringAttribute{
								Optional: true,
							},
							"stacked": schema.BoolAttribute{
								Optional: true,
							},
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.AtLeastOneOf(
						path.MatchRoot("heatmap"),
						path.MatchRoot("list"),
						path.MatchRoot("table"),
						path.MatchRoot("single_value"),
						path.MatchRoot("text"),
						path.MatchRoot("time_series")),
				},
			},
			"data_options": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"no_data": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"message": schema.StringAttribute{
								Required: true,
							},
							"link_text": schema.StringAttribute{
								Optional: true,
							},
							"link_url": schema.StringAttribute{
								Optional: true,
							},
						},
					},
					"refresh_interval": schema.StringAttribute{
						Optional:   true,
						CustomType: fwtypes.TimeRangeType{},
					},
					"hide_missing_values": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
					"max_precision": schema.Int64Attribute{
						Optional: true,
					},
					"time_range": schema.StringAttribute{
						Optional:   true,
						CustomType: fwtypes.TimeRangeType{},
					},
					"unit_prefix": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("Metric"),
						Validators: []validator.String{
							stringvalidator.OneOf("Metric", "Binary"),
						},
					},
				},
			},
			"publish_options": schema.ListNestedAttribute{
				Description: "Allows a user to configure how each published stream on the chart is rendered",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Required: true,
						},
						"display_name": schema.StringAttribute{
							Optional: true,
						},
						"value_prefix": schema.StringAttribute{
							Optional: true,
						},
						"value_suffix": schema.StringAttribute{
							Optional: true,
						},
						"value_unit": schema.StringAttribute{
							Optional: true,
						},
						"plot_type": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (rc *ResourceChart) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model ResourceChartModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}
}

func (rc *ResourceChart) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model ResourceChartModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}
}

func (rc *ResourceChart) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model ResourceChartModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}
}

func (rc *ResourceChart) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model ResourceChartModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}
}
