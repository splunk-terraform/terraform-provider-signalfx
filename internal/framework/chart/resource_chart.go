// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwchart

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	fwvalidator "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/validator"
)

type ResourceChart struct {
	fwembed.ResourceData
}

var (
	_ resource.Resource                     = &ResourceChart{}
	_ resource.ResourceWithConfigure        = &ResourceChart{}
	_ resource.ResourceWithConfigValidators = &ResourceChart{}
	// _ resource.ResourceWithImportState      = &ResourceChart{}
)

func NewResourceChart() resource.Resource {
	return &ResourceChart{}
}

func (rc *ResourceChart) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart"
}

func (rc *ResourceChart) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	sb := NewSchemaBuilder()

	resp.Schema = schema.Schema{
		Description: "The resources defines all the configurable chart types within Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the chart.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the chart.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the chart.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the chart.",
				Optional:    true,
			},
			"tags": schema.ListAttribute{
				Description: "A list of tags associated with the chart.",
				ElementType: types.StringType,
				Computed:    true,
				Optional:    true,
			},
			"heatmap": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.WithCommonChartBase().
					WithOptionalMinResolution().
					WithOptionalGroupBy().
					WithOptionalSortBy().
					WithOptionalColorRange().
					WithOptionalColorScale().
					WithOptionalHideTimeStamps().
					Build(),
			},
			"list": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.WithCommonChartBase().
					WithOptionalHideMissingValues().
					WithOptionalSortBy().
					WithOptionalRefreshInterval().
					WithOptionalLegend().
					WithOptionalMaxPrecision().
					WithOptionalSecondaryVisualization().
					WithOptionalColorScale().
					WithOptionalVizualisationOptions().
					WithOptionalTimeRange().
					WithOptionalStartEndTime().
					Build(),
			},
			"single_value": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.WithCommonChartBase().
					WithOptionalMaxPrecision().
					WithOptionalHideTimeStamps().
					WithOptionalShowSparkLine().
					WithOptionalSecondaryVisualization().
					WithOptionalColorScale().
					WithOptionalVizualisationOptions().
					Build(),
			},
			"table": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.WithCommonChartBase().
					WithOptionalGroupBy().
					WithOptionalHideTimeStamps().
					WithOptionalVizualisationOptions().
					Build(),
			},
			"time": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.
					WithCommonChartBase().
					WithOptionalTimeRange().
					WithOptionalStartEndTime().
					WithOptionalAxes().
					WithOptionalLegend().
					WithOptionalShowEventLines().
					WithOptionalShowDataMarkers().
					WithOptionalStacked().
					WithOptionalPlotType().
					WithOptionalHistogram().
					WithOptionalVizualisationOptions().
					WithOptionalEventOptions().
					Build(),
			},
			"event_feed": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.
					WithRequiredProgramText().
					WithOptionalTimeRange().
					WithOptionalStartEndTime().
					Build(),
			},
			"text": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.
					WithRequiredMarkdown().
					Build(),
			},
			"slo": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: sb.
					WithRequiredSLOID().
					Build(),
			},
		},
	}
}
func (rc *ResourceChart) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Implement validation logic here if needed
}

func (rc *ResourceChart) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Implement validation logic here if needed
}

func (rc *ResourceChart) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Implement validation logic here if needed
}

func (rc *ResourceChart) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Implement validation logic here if needed
}

func (rc *ResourceChart) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		fwvalidator.NewResourceExactlyOnce(
			path.MatchRoot("event_feed"),
			path.MatchRoot("heatmap"),
			path.MatchRoot("list"),
			path.MatchRoot("single_value"),
			path.MatchRoot("slo"),
			path.MatchRoot("table"),
			path.MatchRoot("text"),
			path.MatchRoot("time"),
		),
	}
}
