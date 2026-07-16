// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

const dimensionValuesPageSize = 1000

type DataSourceDimensionValues struct {
	fwembed.DatasourceData
}

type dataSourceDimensionValuesModel struct {
	ID      types.String `tfsdk:"id"`
	Query   types.String `tfsdk:"query"`
	OrderBy types.String `tfsdk:"order_by"`
	Limit   types.Int64  `tfsdk:"limit"`
	Values  types.List   `tfsdk:"values"`
}

var (
	_ datasource.DataSource              = (*DataSourceDimensionValues)(nil)
	_ datasource.DataSourceWithConfigure = (*DataSourceDimensionValues)(nil)
)

func NewDataSourceDimensionValues() datasource.DataSource {
	return &DataSourceDimensionValues{}
}

func (dimensions *DataSourceDimensionValues) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dimension_values"
}

func (dimensions *DataSourceDimensionValues) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries metric dimension values that match a Splunk Observability Cloud metadata expression.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"query": schema.StringAttribute{
				Required:    true,
				Description: "Metric metadata query used to select dimension values.",
			},
			"order_by": schema.StringAttribute{
				Optional:    true,
				Description: "API field and direction used to order matching dimension values.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of matching dimension values to return.",
				Validators:  []validator.Int64{int64validator.Between(0, 10_000)},
			},
			"values": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ordered dimension values matching the configured query.",
			},
		},
	}
}

func (dimensions *DataSourceDimensionValues) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourceDimensionValuesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.Limit.IsNull() || model.Limit.IsUnknown() {
		model.Limit = types.Int64Value(dimensionValuesPageSize)
	}
	totalLimit := int(model.Limit.ValueInt64())
	values := make([]string, 0, totalLimit)
	matchedCount := 0
	for offset := 0; offset < totalLimit; {
		pageLimit := min(dimensionValuesPageSize, totalLimit-offset)
		result, err := dimensions.Details().Client.SearchDimension(
			ctx, model.Query.ValueString(), model.OrderBy.ValueString(), pageLimit, offset,
		)
		if err != nil {
			resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
			return
		}
		if result == nil {
			break
		}
		matchedCount = max(matchedCount, int(result.Count))
		for _, dimension := range result.Results {
			if dimension != nil && len(values) < totalLimit {
				values = append(values, dimension.Value)
			}
		}
		if len(result.Results) == 0 || offset+len(result.Results) >= int(result.Count) || len(values) >= totalLimit {
			break
		}
		offset += pageLimit
	}

	model.ID = types.StringValue(model.Query.ValueString())
	value, diagnostics := types.ListValueFrom(ctx, types.StringType, values)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Values = value
	if matchedCount > totalLimit {
		resp.Diagnostics.AddWarning(
			"Matching dimension values were truncated",
			"The number of matching dimension values exceeds the configured limit. Increase limit or make the query more selective to return every match.",
		)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
