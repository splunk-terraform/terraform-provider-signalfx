// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

type DataSourcePagerDuty struct {
	fwembed.DatasourceData
}

type dataSourcePagerDutyModel struct {
	integrationModel
}

var (
	_ datasource.DataSource              = (*DataSourcePagerDuty)(nil)
	_ datasource.DataSourceWithConfigure = (*DataSourcePagerDuty)(nil)
)

func NewDataSourcePagerDuty() datasource.DataSource {
	return &DataSourcePagerDuty{}
}

func (pagerDuty *DataSourcePagerDuty) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pagerduty_integration"
}

func (pagerDuty *DataSourcePagerDuty) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a PagerDuty integration by its configured name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the PagerDuty integration.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Configured name of the PagerDuty integration.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the PagerDuty integration is enabled.",
			},
		},
	}
}

func (pagerDuty *DataSourcePagerDuty) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourcePagerDutyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := pagerDuty.Details().Client.GetPagerDutyIntegrationByName(ctx, model.Name.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	model.updateWithID(details.Id, details.Name, details.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
