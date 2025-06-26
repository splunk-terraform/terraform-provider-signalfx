// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/autoarchiveexemptmetric"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/autoarchivesettings"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/detector"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/dimension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/organization"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/team"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/track"
	"github.com/splunk-terraform/terraform-provider-signalfx/version"
)

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_token": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"email", "password", "organization_id"},
				DefaultFunc:   schema.EnvDefaultFunc("SFX_AUTH_TOKEN", ""),
				Description:   "Splunk Observability Cloud auth token",
			},
			"api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_API_URL", "https://api.signalfx.com"),
				Description: "API URL for your Splunk Observability Cloud org, may include a realm",
			},
			"custom_app_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_CUSTOM_APP_URL", "https://app.signalfx.com"),
				Description: "Application URL for your Splunk Observability Cloud org, often customized for organizations using SSO",
			},
			"timeout_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     120,
				Description: "Timeout duration for a single HTTP call in seconds. Defaults to 120",
			},
			"retry_max_attempts": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     4,
				Description: "Max retries for a single HTTP call. Defaults to 4",
			},
			"retry_wait_min_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Minimum retry wait for a single HTTP call in seconds. Defaults to 1",
			},
			"retry_wait_max_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     30,
				Description: "Maximum retry wait for a single HTTP call in seconds. Defaults to 30",
			},
			"email": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"auth_token"},
				Description:   "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
			},
			"password": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"auth_token"},
				Description:   "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
			},
			"organization_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"auth_token"},
				Description:   "Required if the user is configured to be part of multiple organizations",
			},
			"feature_preview": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
				Optional:    true,
				Description: "Allows for users to opt-in to new features that are considered experimental or not ready for general availability yet.",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				Optional:    true,
				Description: "Allows for Tags to be added by default to resources that allow for tags to be included. If there is already tags configured, the global tags are added in prefix.",
			},
			"teams": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				Optional:    true,
				Description: "Allows for teams to be defined at a provider level, and apply to all applicable resources created.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			team.ResourceName:                    team.NewResource(),
			detector.ResourceName:                detector.NewResource(),
			autoarchivesettings.ResourceName:     autoarchivesettings.NewResource(),
			autoarchiveexemptmetric.ResourceName: autoarchiveexemptmetric.NewResource(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			dimension.DataSourceName:    dimension.NewDataSource(),
			organization.DataSourceName: organization.NewDataSource(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, data *schema.ResourceData) (any, diag.Diagnostics) {
	meta := &pmeta.Meta{
		Email:          data.Get("email").(string),
		Password:       data.Get("password").(string),
		OrganizationID: data.Get("organization_id").(string),
		Tags:           convert.SliceAll(data.Get("tags").([]any), convert.ToString),
		Teams:          convert.SliceAll(data.Get("teams").([]any), convert.ToString),
	}

	for _, lookup := range pmeta.NewDefaultProviderLookups() {
		if err := lookup.Do(ctx, meta); err != nil {
			tflog.Debug(
				ctx,
				"Issue trying to load external provider configuration, skipping",
				tfext.ErrorLogFields(err),
			)
		}
	}

	if token, ok := data.GetOk("auth_token"); ok {
		meta.AuthToken = token.(string)
	}
	if url, ok := data.GetOk("api_url"); ok {
		meta.APIURL = url.(string)
	}
	if url, ok := data.GetOk("custom_app_url"); ok {
		meta.CustomAppURL = url.(string)
	}

	err := meta.Validate()
	if err != nil {
		return nil, tfext.AsErrorDiagnostics(err)
	}

	var (
		attempts = data.Get("retry_max_attempts").(int)
		timeout  = time.Duration(int64(data.Get("timeout_seconds").(int))) * time.Second
		waitmin  = time.Duration(int64(data.Get("retry_wait_min_seconds").(int))) * time.Second
		waitmax  = time.Duration(int64((data.Get("retry_wait_max_seconds").(int)))) * time.Second
	)

	token, err := meta.LoadSessionToken(ctx)
	if err != nil {
		return nil, tfext.AsErrorDiagnostics(err)
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
		signalfx.UserAgent(fmt.Sprintf("Terraform terraform-provider-signalfx/%s", version.ProviderVersion)),
	)

	if err != nil {
		return nil, tfext.AsErrorDiagnostics(err)
	}

	tflog.Debug(ctx, "Configured settings for http client", tfext.NewLogFields().
		Field("attempts", attempts).
		Duration("timeout", timeout).
		Duration("wait_min", waitmin).
		Duration("wait_max", waitmax),
	)

	for feat, val := range data.Get("feature_preview").(map[string]any) {
		if err := pmeta.LoadPreviewRegistry(ctx, meta).Configure(ctx, feat, val.(bool)); err != nil {
			return nil, tfext.AsWarnDiagnostics(err)
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

	return meta, nil
}
