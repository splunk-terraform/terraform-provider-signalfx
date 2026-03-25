// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package builtincontent

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

type AutoDetectorDataSource struct {
	fwembed.DatasourceData
}

type AutoDetectorModelDataSource struct {
	Results types.Map `tfsdk:"results"`
}

type AutoDetectorIdentifierModelDataSource struct {
	ID     types.String `tfsdk:"id"`
	Inputs types.List   `tfsdk:"inputs"`
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
			"results": schema.MapNestedAttribute{
				Description: "Contains a map of existing auto detector names to their IDs. " +
					"Note that the names are cleaned to be Terraform compatible, so they may differ from the actual auto detector names in Splunk.",
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the auto detector.",
							Computed:    true,
						},
						"inputs": schema.ListAttribute{
							Description: "The values that can be configured as part of the auto detector",
							Computed:    true,
							ElementType: types.MapType{ElemType: types.StringType},
						},
					},
				},
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

	u, _ := url.Parse(dd.Details().APIURL)
	var (
		pageSize  = 100
		results   = make(map[string]*AutoDetectorIdentifierModelDataSource)
		inspector = experimental.NewInspector(u, dd.Details().AuthToken)
	)

	for offset := 0; ; offset += pageSize {
		result, err := client.SearchDetectors(ctx, pageSize, "", offset, "")
		if err != nil {
			resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
			return
		}

		for _, r := range result.Results {
			if r.DetectorOrigin != "AutoDetect" {
				continue
			}

			values, _, err := inspector.GetAutoDetectorArgumentsAndFilters(ctx, r.ProgramText)
			if err != nil {
				tflog.Info(ctx, "Unable to load input details, skipping...", tfext.NewLogFields().
					Error(err).
					Field("detector-id", r.Id).
					Field("detector-name", r.Name).
					Field("program-text", "\""+r.ProgramText+"\""),
				)
				continue
			}
			mapped := []map[string]attr.Value{}
			for _, k := range values {
				mapped = append(mapped, map[string]attr.Value{
					k.Name: types.StringValue(fmt.Sprint(k.DefaultValue)),
				})
			}
			inputs, diag := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, mapped)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}
			results[fwshared.NewCompatibleIdentifer(r.Name)] = &AutoDetectorIdentifierModelDataSource{
				ID:     types.StringValue(r.Id),
				Inputs: inputs,
			}
		}

		if len(result.Results) < pageSize {
			break
		}
	}

	var (
		model      AutoDetectorModelDataSource
		definition = types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":     types.StringType,
				"inputs": types.ListType{ElemType: types.MapType{ElemType: types.StringType}},
			},
		}
	)

	if data, diags := types.MapValueFrom(ctx, definition, results); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	} else {
		model.Results = data
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
