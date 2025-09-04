// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package apm

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
)

type DatasourceTopology struct {
	fwembed.DatasourceData
}

type DatasourceTopologyModel struct {
	StartTime   types.String                     `tfsdk:"start_time"`
	EndTime     types.String                     `tfsdk:"end_time"`
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
	ServiceName types.String `tfsdk:"service_name"`
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

func (dt *DatasourceTopology) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apm_topology"
}

func (dt *DatasourceTopology) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Service Topology allows you to gather data about the service dependencies based on their APM tracing data.",
		Attributes: map[string]schema.Attribute{
			"start_time": schema.StringAttribute{
				Required:    true,
				Description: "Allows the user to set a relative time window as to how far back should be queried to build the service topology",
			},
			"end_time": schema.StringAttribute{
				Optional:    true,
				Description: "Allows to set the exact time window for gathering existing services running",
			},
			"filters": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"scope": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("GLOBAL", "TIER", "INCOMING", "SPAN"),
							},
						},
						"exactly": schema.ListAttribute{
							Description: "Exactly ensures that the key value explicitly matches the provided value, and sets the operator to equals",
							Optional:    true,
							ElementType: types.StringType,
						},
						"matches": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Validators: []validator.Object{
						objectvalidator.ExactlyOneOf(
							path.MatchRoot("exactly"),
							path.MatchRoot("matches"),
						),
					},
				},
			},
			"nodes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of services to include in the topology query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"service_name": schema.StringAttribute{
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
				Computed: true,
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
	dt.DatasourceData.Details().Client.Top
}
