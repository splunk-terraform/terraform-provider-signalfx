// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/go-homedir"
	sfx "github.com/signalfx/signalfx-go"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/autoarchivesettings"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/organization"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/track"
	"github.com/splunk-terraform/terraform-provider-signalfx/version"
)

var SystemConfigPath = "/etc/signalfx.conf"
var HomeConfigSuffix = "/.signalfx.conf"
var HomeConfigPath = ""

var sfxProvider *schema.Provider

type signalfxConfig = pmeta.Meta

func Provider() *schema.Provider {
	sfxProvider = &schema.Provider{
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
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
				RequiredWith: []string{
					"password",
					"organization_id",
				},
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Used to create a session token instead of an API token, it requires the account to be configured to login with Email and Password",
				RequiredWith: []string{
					"email",
					"organization_id",
				},
			},
			"organization_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Required if the user is configured to be part of multiple organizations",
			},
			"feature_preview": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
				Optional:    true,
				Description: "Allows for users to opt-in to new features that are considered experimental or not ready for general availabilty yet.",
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
		DataSourcesMap: map[string]*schema.Resource{
			"signalfx_dimension_values":      dataSourceDimensionValues(),
			"signalfx_pagerduty_integration": dataSourcePagerDutyIntegration(),
			organization.DataSourceName:      organization.NewDataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"signalfx_alert_muting_rule":           alertMutingRuleResource(),
			"signalfx_automated_archival_settings": autoarchivesettings.NewResource(),
			"signalfx_aws_external_integration":    integrationAWSExternalResource(),
			"signalfx_aws_token_integration":       integrationAWSTokenResource(),
			"signalfx_aws_integration":             integrationAWSResource(),
			"signalfx_azure_integration":           integrationAzureResource(),
			"signalfx_dashboard":                   dashboardResource(),
			"signalfx_dashboard_group":             dashboardGroupResource(),
			"signalfx_data_link":                   dataLinkResource(),
			"signalfx_detector":                    detectorResource(),
			"signalfx_event_feed_chart":            eventFeedChartResource(),
			"signalfx_gcp_integration":             integrationGCPResource(),
			"signalfx_heatmap_chart":               heatmapChartResource(),
			"signalfx_jira_integration":            integrationJiraResource(),
			"signalfx_list_chart":                  listChartResource(),
			"signalfx_org_token":                   orgTokenResource(),
			"signalfx_opsgenie_integration":        integrationOpsgenieResource(),
			"signalfx_pagerduty_integration":       integrationPagerDutyResource(),
			"signalfx_service_now_integration":     integrationServiceNowResource(),
			"signalfx_slack_integration":           integrationSlackResource(),
			"signalfx_single_value_chart":          singleValueChartResource(),
			"signalfx_slo_chart":                   sloChartResource(),
			"signalfx_team":                        teamResource(),
			"signalfx_time_chart":                  timeChartResource(),
			"signalfx_text_chart":                  textChartResource(),
			"signalfx_victor_ops_integration":      integrationVictorOpsResource(),
			"signalfx_webhook_integration":         integrationWebhookResource(),
			"signalfx_log_view":                    logViewResource(),
			"signalfx_log_timeline":                logTimelineResource(),
			"signalfx_table_chart":                 tableChartResource(),
			"signalfx_metric_ruleset":              metricRulesetResource(),
			"signalfx_slo":                         sloResource(),
		},
		ConfigureFunc: signalfxConfigure,
	}

	return sfxProvider
}

func signalfxConfigure(data *schema.ResourceData) (interface{}, error) {
	config := signalfxConfig{
		Email:          data.Get("email").(string),
		Password:       data.Get("password").(string),
		OrganizationID: data.Get("organization_id").(string),
		Tags:           convert.SliceAll(data.Get("tags").([]any), convert.ToString),
		Teams:          convert.SliceAll(data.Get("teams").([]any), convert.ToString),
	}

	// /etc/signalfx.conf has the lowest priority
	if _, err := os.Stat(SystemConfigPath); err == nil {
		err = readConfigFile(SystemConfigPath, &config)
		if err != nil {
			return nil, err
		}
	}

	// $HOME/.signalfx.conf second
	// this additional variable is used for mocking purposes in tests
	if HomeConfigPath == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get user environment %s", err.Error())
		}
		HomeConfigPath = usr.HomeDir + HomeConfigSuffix
	}
	if _, err := os.Stat(HomeConfigPath); err == nil {
		err = readConfigFile(HomeConfigPath, &config)
		if err != nil {
			return nil, err
		}
	}

	// Use netrc next
	err := readNetrcFile(&config)
	if err != nil {
		return nil, err
	}

	// provider is the top priority
	if token, ok := data.GetOk("auth_token"); ok {
		config.AuthToken = token.(string)
	}

	if url, ok := data.GetOk("api_url"); ok {
		config.APIURL = url.(string)
	}
	if customAppURL, ok := data.GetOk("custom_app_url"); ok {
		config.CustomAppURL = customAppURL.(string)
	}

	if err = config.Validate(); err != nil {
		return nil, err
	}

	netTransport := logging.NewTransport("SignalFx", &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	})

	pv := version.ProviderVersion
	providerUserAgent := fmt.Sprintf("Terraform/%s terraform-provider-signalfx/%s", sfxProvider.TerraformVersion, pv)

	totalTimeoutSeconds := data.Get("timeout_seconds").(int)
	retryMaxAttempts := data.Get("retry_max_attempts").(int)
	retryWaitMinSeconds := data.Get("retry_wait_min_seconds").(int)
	retryWaitMaxSeconds := data.Get("retry_wait_max_seconds").(int)
	log.Printf("[DEBUG] SignalFx: HTTP Timeout is %d seconds", totalTimeoutSeconds)
	log.Printf("[DEBUG] SignalFx: HTTP max retry attempts: %d", retryMaxAttempts)
	log.Printf("[DEBUG] SignalFx: HTTP retry wait min is %d seconds", retryWaitMinSeconds)
	log.Printf("[DEBUG] SignalFx: HTTP retry wait max is %d seconds", retryWaitMaxSeconds)

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retryMaxAttempts
	retryClient.RetryWaitMin = time.Second * time.Duration(int64(retryWaitMinSeconds))
	retryClient.RetryWaitMax = time.Second * time.Duration(int64(retryWaitMaxSeconds))
	retryClient.HTTPClient.Timeout = time.Second * time.Duration(int64(totalTimeoutSeconds))
	retryClient.HTTPClient.Transport = netTransport
	standardClient := retryClient.StandardClient()

	token, err := config.LoadSessionToken(context.Background())
	if err != nil {
		return nil, err
	}

	client, err := sfx.NewClient(
		token,
		sfx.APIUrl(config.APIURL),
		sfx.HTTPClient(standardClient),
		sfx.UserAgent(providerUserAgent),
	)
	if err != nil {
		return &config, err
	}

	config.Client = client

	for feat, val := range data.Get("feature_preview").(map[string]any) {
		err = pmeta.LoadPreviewRegistry(
			context.TODO(),
			config,
		).Configure(context.TODO(), feat, val.(bool))

		if err != nil {
			return nil, err
		}
	}

	if gate, ok := pmeta.LoadPreviewRegistry(context.TODO(), config).Get(feature.PreviewProviderTracking); ok && gate.Enabled() {
		tracking, err := track.ReadGitDetails(context.TODO())
		if err != nil {
			log.Printf("[INFO] Unable to load git details, skipping: %v", err)
			tflog.Info(context.TODO(), "Unable to load git details, skipping", tfext.ErrorLogFields(err))
		} else {
			config.Tags = append(config.Tags, tracking.Tags()...)
		}
	}

	return &config, nil
}

func readConfigFile(configPath string, config *signalfxConfig) error {
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file. %s", err.Error())
	}
	err = json.Unmarshal(configFile, config)
	if err != nil {
		return fmt.Errorf("failed to parse config file. %s", err.Error())
	}
	return nil
}

func readNetrcFile(config *signalfxConfig) error {
	// Inspired by https://github.com/hashicorp/terraform/blob/master/vendor/github.com/hashicorp/go-getter/netrc.go
	// Get the netrc file path
	path := os.Getenv("NETRC")
	if path == "" {
		filename := ".netrc"
		if runtime.GOOS == "windows" {
			filename = "_netrc"
		}

		var err error
		path, err = homedir.Expand("~/" + filename)
		if err != nil {
			return err
		}
	}

	// If the file is not a file, then do nothing
	if fi, err := os.Stat(path); err != nil {
		// File doesn't exist, do nothing
		if os.IsNotExist(err) {
			return nil
		}

		// Some other error!
		return err
	} else if fi.IsDir() {
		// File is directory, ignore
		return nil
	}

	// Load up the netrc file
	netRC, err := netrc.ParseFile(path)
	if err != nil {
		return fmt.Errorf("error parsing netrc file at %q: %s", path, err)
	}

	machine := netRC.FindMachine("api.signalfx.com")
	if machine == nil {
		// Machine not found, no problem
		return nil
	}

	// Set the auth token
	config.AuthToken = machine.Password
	return nil
}
