// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwchart

import "github.com/hashicorp/terraform-plugin-framework/types"

type ResourceChartModel struct {
	ID          types.String                `tfsdk:"id"`
	URL         types.String                `tfsdk:"url"`
	Name        types.String                `tfsdk:"name"`
	Description types.String                `tfsdk:"description"`
	Tags        types.List                  `tfsdk:"tags"`
	EventFeed   *ChartKindEventFeedOption   `tfsdk:"event_feed"`
	Heatmap     *ChartKindHeatmapOption     `tfsdk:"heatmap"`
	List        *ChartKindListOption        `tfsdk:"list"`
	SingleValue *ChartKindSingleValueOption `tfsdk:"single_value"`
	SLO         *ChartKindSLOOption         `tfsdk:"slo"`
	Table       *ChartKindTableOption       `tfsdk:"table"`
	Text        *ChartKindTextOptions       `tfsdk:"text"`
	Time        *ChartKindTimeOption        `tfsdk:"time"`
}

type ChartKindTextOptions struct {
	Markdown types.String `tfsdk:"markdown"`
}

type ChartKindSLOOption struct {
	SLOID types.String `tfsdk:"slo_id"`
}

type ChartKindTimeOption struct{}

type ChartKindEventFeedOption struct{}

type ChartKindHeatmapOption struct{}

type ChartKindListOption struct{}

type ChartKindSingleValueOption struct{}

type ChartKindTableOption struct{}
