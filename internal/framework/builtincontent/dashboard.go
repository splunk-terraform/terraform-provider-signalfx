// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package builtincontent

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/sync/errgroup"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

type DashboardGroupsDataSource struct {
	fwembed.DatasourceData
	replacer *regexp.Regexp
}

type DashboardGroupsModelDataSource struct {
	Results types.Map `tfsdk:"results"`
}

var (
	_ datasource.DataSource              = (*DashboardGroupsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*DashboardGroupsDataSource)(nil)
)

func NewDashboardGroupsDataSource() datasource.DataSource {
	return &DashboardGroupsDataSource{
		replacer: regexp.MustCompile(`(?<empty>[^\w]+)`),
	}
}

func (dg *DashboardGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_builtin_dashboards"
}

func (dg *DashboardGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source is responsible for capturing all the built in content available for the user so that they can be used within their own dashboard groups.",
		Attributes: map[string]schema.Attribute{
			"results": schema.MapAttribute{
				Description: "Map of the builtin content dashboard groups and their associated dashboards. " +
					"The keys are the dashboard group names and the values are maps of dashboard names to their IDs.",
				Computed: true,
				ElementType: types.MapType{
					ElemType: types.StringType,
				},
			},
		},
	}
}

func (dg *DashboardGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := pmeta.LoadClient(ctx, dg.Details())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Load Client", err.Error())
		return
	}

	var (
		wg = errgroup.Group{}

		mu       sync.Mutex
		resolved = make(map[string]map[string]string)
	)
	wg.SetLimit(10)

	for offset, limit := 0, 100; ; offset += limit {
		results, err := client.ListBuiltInDashboardGroups(ctx, limit, offset)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Fetch Dashboard Groups",
				err.Error(),
			)
			return
		}
		if len(results.Results) == 0 {
			break
		}
		for _, r := range results.Results {
			for _, dashboard := range r.Dashboards {
				group := dg.clean(r.Name)
				resolved[group] = make(map[string]string)
				wg.Go(func() error {
					result, err := client.GetDashboard(ctx, dashboard)
					if err != nil {
						return err
					}
					mu.Lock()
					resolved[group][dg.clean(result.Name)] = dashboard
					mu.Unlock()
					return nil
				})
			}
		}
	}

	if err := wg.Wait(); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Resolve Dashboard Groups",
			err.Error(),
		)
		return
	}

	results, diags := types.MapValueFrom(ctx, types.MapType{ElemType: types.StringType}, resolved)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	model := &DashboardGroupsModelDataSource{
		Results: results,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (df *DashboardGroupsDataSource) clean(name string) string {
	return strings.Trim(df.replacer.ReplaceAllString(name, "_"), "_")
}
