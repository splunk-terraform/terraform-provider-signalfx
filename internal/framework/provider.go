// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/signalfx/signalfx-go"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/apm"
	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	internalfunction "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/function"
	fwintegration "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/integration"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/track"
)

type ollyProvider struct {
	version  string
	features *feature.Registry
}

var (
	_ provider.Provider                   = (*ollyProvider)(nil)
	_ provider.ProviderWithFunctions      = (*ollyProvider)(nil)
	_ provider.ProviderWithValidateConfig = (*ollyProvider)(nil)
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
	if pv, err := version.NewVersion(req.TerraformVersion); err != nil {
		tflog.Debug(ctx, "Unable to parse the terraform version used", tfext.ErrorLogFields(err))
	} else if pv.LessThan(version.Must(version.NewVersion("1.11.0"))) {
		resp.Diagnostics.AddWarning(
			"Unsupported Terraform Version",
			"In version 10.x of the SignalFx provider, the framework is adopting features that require the installed terraform version to be greater than 1.11.0 to function properly.\n"+
				"Please prepare to migrate to a newer version of terraform soon.",
		)
	}

	model := newDefaultOllyProviderModel()

	if errs := req.Config.Get(ctx, model); errs.HasError() {
		// The original code uses unexposed types that make it hard to validate
		// however, this path should not be possible to hit in typical usage.
		resp.Diagnostics.AddAttributeError(
			path.Empty(),
			"Internal Provider Error",
			"An expected error occurred while configuring the provider. Please report this issue to the provider developers.",
		)
		tflog.Error(ctx, "Error configuring provider", tfext.NewLogFields().Field("errors", errs))
		return
	}

	model.init()

	meta := &pmeta.Meta{
		Registry:       op.features,
		Email:          model.Email.ValueString(),
		Password:       model.Password.ValueString(),
		OrganizationID: model.OrganizationID.ValueString(),
	}

	for _, val := range model.Tags.Elements() {
		if tag, ok := val.(types.String); ok && !tag.IsNull() {
			meta.Tags = append(meta.Tags, tag.ValueString())
		}
	}

	for _, val := range model.Teams.Elements() {
		if team, ok := val.(types.String); ok && !team.IsNull() {
			meta.Teams = append(meta.Teams, team.ValueString())
		}
	}

	for _, lookup := range pmeta.NewDefaultProviderLookups() {
		if err := lookup.Do(ctx, meta); err != nil {
			tflog.Debug(ctx,
				"Issue trying to load external provider configuration, skipping",
				tfext.ErrorLogFields(err),
			)
		}
	}

	if !model.AuthToken.IsNull() {
		meta.AuthToken = model.AuthToken.ValueString()
	}

	if !model.APIURL.IsNull() {
		meta.APIURL = model.APIURL.ValueString()
	}

	if err := meta.Validate(); err != nil {
		resp.Diagnostics.AddError("Issue configuring provider", err.Error())
		return
	}

	var (
		attempts = int(model.RetryMaxAttempts.ValueInt32())
		timeout  = time.Duration(model.TimeoutSeconds.ValueInt64()) * time.Second
		waitmin  = time.Duration(model.RetryWaitMinSeconds.ValueInt64()) * time.Second
		waitmax  = time.Duration(model.RetryWaitMaxSeconds.ValueInt64()) * time.Second
	)

	token, err := meta.LoadSessionToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Issue loading session token", err.Error())
		return
	}

	rc := retryablehttp.NewClient()
	rc.RetryMax = attempts
	rc.RetryWaitMin = waitmin
	rc.RetryWaitMax = waitmax
	rc.HTTPClient.Timeout = timeout
	rc.HTTPClient.Transport = logging.NewSubsystemLoggingHTTPTransport("signalfx", &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	})

	meta.Client, err = signalfx.NewClient(
		token,
		signalfx.APIUrl(meta.APIURL),
		signalfx.HTTPClient(rc.StandardClient()),
		signalfx.UserAgent(fmt.Sprintf("Terraform %s terraform-provider-signalfx/%s", req.TerraformVersion, op.version)),
	)

	if err != nil {
		resp.Diagnostics.AddError("Issue creating Signalfx client", err.Error())
		return
	}

	tflog.Debug(ctx, "Configured settings for http client", tfext.NewLogFields().
		Field("attempts", attempts).
		Duration("timeout", timeout).
		Duration("wait_min", waitmin).
		Duration("wait_max", waitmax),
	)

	if site, err := meta.DetectCustomAPPURL(ctx); err != nil {
		if !model.CustomAppURL.IsNull() {
			meta.CustomAppURL = model.CustomAppURL.ValueString()
		}
	} else {
		meta.CustomAppURL = site
	}

	for name, val := range model.FeaturePreview.Elements() {
		active := val.Equal(types.BoolValue(true))
		if err := pmeta.LoadPreviewRegistry(ctx, meta).Configure(ctx, name, active); err != nil {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("feature_preview").AtMapKey(name),
				"Failed to load feature preview",
				err.Error(),
			)
		}
	}

	if gate, ok := pmeta.LoadPreviewRegistry(ctx, meta).Get(feature.PreviewProviderTracking); ok && gate.Enabled() {
		tracking, err := track.ReadGitDetails(ctx)
		if err != nil {
			tflog.Info(ctx, "Unable to load git details, skipping", tfext.ErrorLogFields(err))
		} else {
			meta.Tags = append(meta.Tags, tracking.Tags()...)
		}
	}

	// Ensure all the data sources are set so they can be consumed by each of the components.
	resp.DataSourceData = meta
	resp.ResourceData = meta
	resp.EphemeralResourceData = meta
}

func (op *ollyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		apm.NewDatasourceTopology,
	}
}

func (op *ollyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		fwintegration.NewResourceSplunkOncall,
		fwdashify.NewResourceDashifyTemplate,
	}
}

func (op *ollyProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		internalfunction.NewTimeRangeParser,
	}
}

func (op *ollyProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	model := newDefaultOllyProviderModel()

	resp.Diagnostics.Append(req.Config.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model.init()

	if model.APIURL.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing API Endpoint",
			"Field must be set to a valid endpoint for the Splunk Observability Cloud provider.",
		)
	}

	switch {
	case !model.AuthToken.IsNull():
		tflog.Debug(ctx, "Using auth token for authentication")
	case !model.Email.IsNull() &&
		!model.Password.IsNull() &&
		!model.OrganizationID.IsNull():
		tflog.Debug(ctx, "Using email and password for authentication")
	default:
		resp.Diagnostics.AddWarning(
			"Missing Authentication Method",
			"Either 'auth_token' or both 'email' and 'password' must be set for authentication as part of the terraform configuration. "+
				"Using external configuration methods will be deprecated in a future major release.",
		)
	}
}
