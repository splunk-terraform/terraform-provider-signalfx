// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package apm

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/signalfx/signalfx-go/apm"
	"github.com/tilinna/clock"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwtypes "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/types"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

type DatasourceTopology struct {
	fwembed.DatasourceData
}

type DatasourceTopologyModel struct {
	StartTime   fwtypes.TimeRange                `tfsdk:"start_time"`
	EndTime     fwtypes.TimeRange                `tfsdk:"end_time"`
	Filters     []*DatasourceTopologyFilterModel `tfsdk:"filters"`
	Services    []*DatasourceTopologyNodeModel   `tfsdk:"nodes"`
	Connections []*DatasourceTopologyEdgeModel   `tfsdk:"edges"`
}

type DatasourceTopologyFilterModel struct {
	Name    types.String `tfsdk:"name"`
	Scope   types.String `tfsdk:"scope"`
	Exactly types.String `tfsdk:"exactly"`
	Matches types.List   `tfsdk:"matches"`
}

type DatasourceTopologyNodeModel struct {
	ServiceName types.String `tfsdk:"name"`
	Inferred    types.Bool   `tfsdk:"inferred"`
	Type        types.String `tfsdk:"type"`
}

type DatasourceTopologyEdgeModel struct {
	From types.String `tfsdk:"from"`
	To   types.String `tfsdk:"to"`
}

var (
	_ datasource.DataSource              = (*DatasourceTopology)(nil)
	_ datasource.DataSourceWithConfigure = (*DatasourceTopology)(nil)
)

func NewDatasourceTopology() datasource.DataSource {
	return &DatasourceTopology{}
}

func (dt *DatasourceTopology) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apm_service_topology"
}

func (dt *DatasourceTopology) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Service Topology allows you to gather data about the service dependencies based on their APM tracing data.",
		Attributes: map[string]schema.Attribute{
			"start_time": schema.StringAttribute{
				Required:    true,
				CustomType:  fwtypes.TimeRangeType{},
				Description: "Start Time is the relative time range of how far to look back when querying for running services that instrumented and reporting data to APM.",
			},
			"end_time": schema.StringAttribute{
				Optional:    true,
				CustomType:  fwtypes.TimeRangeType{},
				Description: "Allows to set the exact time window for gathering existing services running",
			},
			"filters": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Filters allow to narrow down the returned services and their connections based on the provided filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name is the span tag key to filter on",
						},
						"scope": schema.StringAttribute{
							Required:    true,
							Description: "Scope is to set at what level the filter should be applied.",
							MarkdownDescription: "Scope is to set at what level the filter should be applied. Possible values are:\n\n" +
								"- `GLOBAL` - Matches the first occurrence in all spans\n" +
								"- `TIER` - Matches the first occurrence in service-tier spans\n" +
								"- `INCOMING` - Matches the value on the incoming edge span of service tier spans\n" +
								"- `SPAN` - Matches the tag on each span within the trace",
							Validators: []validator.String{
								stringvalidator.OneOf("GLOBAL", "TIER", "INCOMING", "SPAN"),
							},
						},
						"exactly": schema.StringAttribute{
							Optional:    true,
							Description: "Exactly ensures that the key value explicitly matches the provided value, and sets the operator to equals",
						},
						"matches": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Matches allows to provide a list of values to match against",
						},
					},
				},
			},
			"nodes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of services to include in the topology query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the service.",
						},
						"inferred": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the service was inferred from traces.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the service.",
						},
					},
				},
			},
			"edges": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Edges is the table that shows the conntection between services within the topology.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"from": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the service.",
						},
						"to": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the service.",
						},
					},
				},
			},
		},
	}
}

func (dt *DatasourceTopology) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DatasourceTopologyModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	var (
		end = clock.Now(ctx)
	)

	duration, err := model.StartTime.ParseDuration()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Start Time", fmt.Sprintf("The provided start_time %q could not be parsed: %s", model.StartTime, err.Error()))
		return
	}

	if duration < 0 {
		duration = -duration
	}
	begin := end.Add(-duration)

	if !model.EndTime.IsNull() && !model.EndTime.IsUnknown() {
		duration, err = model.EndTime.ParseDuration()
		if err != nil {
			resp.Diagnostics.AddError("Invalid End Time", fmt.Sprintf("The provided end_time %q could not be parsed: %s", model.EndTime, err.Error()))
			return
		}
		if duration < 0 {
			duration = -duration
		}
		end = begin.Add(duration)
	}

	if end.Sub(begin) < 5*time.Minute {
		resp.Diagnostics.AddError(
			"Invalid Time Range",
			"The diference between start_time and end_time must greater than 5 minutes,"+
				"currently it is "+end.Sub(begin).String(),
		)
		return
	}

	tflog.Error(ctx, "Total lookback duration", tfext.NewLogFields().
		Field("begin", begin).
		Field("end", end),
	)
	ask := &apm.RetrieveServiceTopologyRequest{
		TimeRange: fmt.Sprintf("%d/%d", begin.UnixMilli(), end.UnixMilli()),
	}

	for i, ft := range model.Filters {
		if ft == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("filters").AtListIndex(i),
				"Invalid Filter",
				"Filter cannot be null",
			)
			return
		}
		switch {
		case !ft.Exactly.IsNull() && !ft.Exactly.IsUnknown():
			ask.TagFilters = append(ask.TagFilters, apm.TagFiltersInner{
				EqualsTagFilter: &apm.EqualsTagFilter{
					Name:     ft.Name.ValueString(),
					Scope:    ft.Scope.ValueString(),
					Operator: "equals",
					Value:    ft.Exactly.ValueString(),
				},
			})
		case !ft.Matches.IsNull() && !ft.Matches.IsUnknown():
			var matches []string
			if resp.Diagnostics.Append(ft.Matches.ElementsAs(ctx, &matches, false)...); resp.Diagnostics.HasError() {
				return
			}
			ask.TagFilters = append(ask.TagFilters, apm.TagFiltersInner{
				InTagFilter: &apm.InTagFilter{
					Name:     ft.Name.ValueString(),
					Scope:    ft.Scope.ValueString(),
					Operator: "matches",
					Values:   matches,
				},
			})
		}
	}

	details, err := dt.Details().Client.ListTopology(ctx, ask)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}

	for _, node := range details.Data.Nodes {
		model.Services = append(model.Services, &DatasourceTopologyNodeModel{
			ServiceName: types.StringValue(node.GetServiceName()),
			Inferred:    types.BoolValue(node.GetInferred()),
			Type:        types.StringValue(string(node.GetType())),
		})
	}
	for _, edge := range details.Data.Edges {
		model.Connections = append(model.Connections, &DatasourceTopologyEdgeModel{
			From: types.StringPointerValue(edge.FromNode),
			To:   types.StringPointerValue(edge.ToNode),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
