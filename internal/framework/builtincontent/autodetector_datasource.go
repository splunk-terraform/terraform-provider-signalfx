// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package builtincontent

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

type AutoDetectorDataSource struct {
	fwembed.DatasourceData
}

type AutoDetectorModelDataSource struct {
	Results types.Map `tfsdk:"results"`
}

var (
	_ datasource.DataSource              = (*AutoDetectorDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*AutoDetectorDataSource)(nil)
)

func NewAutoDetectorDataSource() datasource.DataSource {
	return &AutoDetectorDataSource{}
}

func (dd *AutoDetectorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auto_detector"
}

func (dd *AutoDetectorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source is used to fetch the existing auto detectors in the organization.",
		Attributes: map[string]schema.Attribute{
			"results": schema.MapAttribute{
				Description: "Contains a map of existing auto detector names to their IDs. " +
					"Note that the names are cleaned to be Terraform compatible, so they may differ from the actual auto detector names in Splunk.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (dd *AutoDetectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := pmeta.LoadClient(ctx, dd.Details())
	if err != nil {
		resp.Diagnostics.AddError("Unable to load client", err.Error())
		return
	}

	var (
		pageSize = 100
		results  = make(map[string]string)
	)

	for offset := 0; ; offset += pageSize {
		result, err := client.SearchDetectors(ctx, pageSize, "", offset, "")
		if err != nil {
			resp.Diagnostics.AddError("Unable to fetch auto detectors", err.Error())
			return
		}

		for _, r := range result.Results {
			if r.DetectorOrigin == "AutoDetect" {
				results[fwshared.NewCompatibleIdentifer(r.Name)] = r.Id
			}
		}

		if len(result.Results) < pageSize {
			break
		}
	}

	var model AutoDetectorModelDataSource

	if data, diags := types.MapValueFrom(ctx, types.StringType, results); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	} else {
		model.Results = data
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
