// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtimestamp

import (
	"context"
	"time"
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DatasourceTimestamp struct{}

type DatasourceTimestampModel struct {
	Timezone types.String `tfsdk:"timezone"`
	Year     types.Int32  `tfsdk:"year"`
	Month    types.Int32  `tfsdk:"month"`
	Day      types.Int32  `tfsdk:"day"`
	Hour     types.Int32  `tfsdk:"hour"`
	Minute   types.Int32  `tfsdk:"minute"`
	Second   types.Int32  `tfsdk:"second"`

	Value types.Int64 `tfsdk:"value"`
}

var (
	_ datasource.DataSource = (*DatasourceTimestamp)(nil)
)

func NewDatasourceTimestamp() datasource.DataSource {
	return &DatasourceTimestamp{}
}

func (DatasourceTimestamp) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_timestamp"
}

func (DatasourceTimestamp) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Converts a date and time in a specified timezone to a Unix timestamp in milliseconds.",
		Attributes: map[string]schema.Attribute{
			"timezone": schema.StringAttribute{
				Optional:    true,
				Description: "IANA timezone name used to interpret the date and time, for example Europe/London or UTC.",
			},
			"year": schema.Int32Attribute{
				Required:    true,
				Description: "Calendar year, from 1 to 9999.",
				Validators: []validator.Int32{
					int32validator.Between(1, 9999),
				},
			},
			"month": schema.Int32Attribute{
				Optional:    true,
				Description: "Month of the year, from 1 (January) to 12 (December).",
				Validators: []validator.Int32{
					int32validator.Between(1, 12),
				},
			},
			"day": schema.Int32Attribute{
				Optional:    true,
				Description: "Day of the month, from 0 to 31.",
				Validators: []validator.Int32{
					int32validator.Between(0, 31),
				},
			},
			"hour": schema.Int32Attribute{
				Optional:    true,
				Description: "Hour of the day, from 0 to 23.",
				Validators: []validator.Int32{
					int32validator.Between(0, 23),
				},
			},
			"minute": schema.Int32Attribute{
				Optional:    true,
				Description: "Minute of the hour, from 0 to 59.",
				Validators: []validator.Int32{
					int32validator.Between(0, 59),
				},
			},
			"second": schema.Int32Attribute{
				Optional:    true,
				Description: "Second of the minute, from 0 to 59.",
				Validators: []validator.Int32{
					int32validator.Between(0, 59),
				},
			},
			"value": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp for the configured date and time, in milliseconds.",
			},
		},
	}
}

func (DatasourceTimestamp) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DatasourceTimestampModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	loc, err := time.LoadLocation(model.Timezone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to parse timezone", err.Error())
		return
	}

	month := time.January
	if num := time.Month(model.Month.ValueInt32()); num >= time.January && num <= time.December {
		month = num
	}

	ts := time.Date(
		int(model.Year.ValueInt32()),
		month,
		int(model.Day.ValueInt32()),
		int(model.Hour.ValueInt32()),
		int(model.Minute.ValueInt32()),
		int(model.Second.ValueInt32()),
		0,
		loc,
	)

	model.Value = types.Int64Value(ts.UnixMilli())
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
