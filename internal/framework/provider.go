// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	internalfunction "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/function"
)

type ollyProvider struct {
	version  string
	features *feature.Registry
}

var (
	_ provider.Provider              = (*ollyProvider)(nil)
	_ provider.ProviderWithFunctions = (*ollyProvider)(nil)
)

func NewProvider(version string, opts ...ProviderOption) provider.Provider {
	op := &ollyProvider{
		version:  version,
		features: feature.GetGlobalRegistry(),
	}
	for _, opt := range opts {
		opt(op)
	}
	return op
}

func (op *ollyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "signalfx"
	resp.Version = op.version
}

func (op *ollyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	// Note (MovieStoreGuy): While we transition away from the V2 SDK, this will need to match the older provider schema.
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"auth_token": schema.StringAttribute{
				Optional:    true,
				Description: "Splunk Observability Cloud auth token",
			},
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "API URL for your Splunk Observability Cloud org, may include a realm",
			},
			"custom_app_url": schema.StringAttribute{
				Optional:           true,
				DeprecationMessage: "Remove the definition, the provider will automatically populate the custom app URL as needed",
				Description:        "Application URL for your Splunk Observability Cloud org, often customized for organizations using SSO",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Timeout duration for a single HTTP call in seconds. Defaults to 120",
			},
			"retry_max_attempts": schema.Int32Attribute{
				Optional:    true,
				Description: "Max retries for a single HTTP call. Defaults to 4",
			},
			"retry_wait_min_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Minimum retry wait for a single HTTP call in seconds. Defaults to 1",
			},
			"retry_wait_max_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum retry wait for a single HTTP call in seconds. Defaults to 30",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Required if the user is configured to be part of multiple organizations",
			},
			"feature_preview": schema.MapAttribute{
				ElementType: types.BoolType,
				Optional:    true,
				Description: "Allows for users to opt-in to new features that are considered experimental or not ready for general availability yet.",
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Allows for Tags to be added by default to resources that allow for tags to be included. If there is already tags configured, the global tags are added in prefix.",
			},
			"teams": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Allows for teams to be defined at a provider level, and apply to all applicable resources created.",
			},
		},
	}
}

func (op *ollyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	model := newDefaultOllyProviderModel()

	diags := req.Config.Get(ctx, model)
	if diags.HasError() {
		resp.Diagnostics = diags
		return
	}

}

func (op *ollyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	// To implement: Register data sources.
	return nil
}

func (op *ollyProvider) Resources(ctx context.Context) []func() resource.Resource {
	// To implement: Register resources.
	return nil
}

func (op *ollyProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		internalfunction.NewTimeRangeParser,
	}
}
