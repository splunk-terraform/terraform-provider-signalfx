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
	"github.com/signalfx/signalfx-go"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/dimensions"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/team"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/version"
)

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_AUTH_TOKEN", ""),
				Description: "Splunk Observability Cloud auth token",
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
		},
		ResourcesMap: map[string]*schema.Resource{
			team.ResourceName: team.NewResource(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			dimensions.DataSourceName: dimensions.NewDataSource(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, data *schema.ResourceData) (any, diag.Diagnostics) {
	var meta pmeta.Meta
	for _, lookup := range pmeta.NewDefaultProviderLookups() {
		if err := lookup.Do(ctx, &meta); err != nil {
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

	meta.Client, err = signalfx.NewClient(meta.AuthToken,
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

	return &meta, nil
}
