package signalfx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	homedir "github.com/mitchellh/go-homedir"
	sfx "github.com/signalfx/signalfx-go"
	"github.com/terraform-providers/terraform-provider-signalfx/version"
)

var SystemConfigPath = "/etc/signalfx.conf"
var HomeConfigSuffix = "/.signalfx.conf"
var HomeConfigPath = ""

var sfxProvider *schema.Provider

type signalfxConfig struct {
	AuthToken    string `json:"auth_token"`
	APIURL       string `json:"api_url"`
	CustomAppURL string `json:"custom_app_url"`
	Client       *sfx.Client
}

func Provider() terraform.ResourceProvider {
	sfxProvider = &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_AUTH_TOKEN", ""),
				Description: "SignalFx auth token",
			},
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_API_URL", "https://api.signalfx.com"),
				Description: "API URL for your SignalFx org, may include a realm",
			},
			"custom_app_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SFX_CUSTOM_APP_URL", "https://app.signalfx.com"),
				Description: "Application URL for your SignalFx org, often customzied for organizations using SSO",
			},
			"timeout_seconds": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     120,
				Description: "Timeout duration for a single HTTP call in seconds. Defaults to 120",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"signalfx_aws_services":     dataSourceAwsServices(),
			"signalfx_azure_services":   dataSourceAzureServices(),
			"signalfx_dimension_values": dataSourceDimensionValues(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"signalfx_alert_muting_rule":        alertMutingRuleResource(),
			"signalfx_aws_external_integration": integrationAWSExternalResource(),
			"signalfx_aws_token_integration":    integrationAWSTokenResource(),
			"signalfx_aws_integration":          integrationAWSResource(),
			"signalfx_azure_integration":        integrationAzureResource(),
			"signalfx_dashboard":                dashboardResource(),
			"signalfx_dashboard_group":          dashboardGroupResource(),
			"signalfx_data_link":                dataLinkResource(),
			"signalfx_detector":                 detectorResource(),
			"signalfx_event_feed_chart":         eventFeedChartResource(),
			"signalfx_gcp_integration":          integrationGCPResource(),
			"signalfx_heatmap_chart":            heatmapChartResource(),
			"signalfx_jira_integration":         integrationJiraResource(),
			"signalfx_list_chart":               listChartResource(),
			"signalfx_org_token":                orgTokenResource(),
			"signalfx_opsgenie_integration":     integrationOpsgenieResource(),
			"signalfx_pagerduty_integration":    integrationPagerDutyResource(),
			"signalfx_slack_integration":        integrationSlackResource(),
			"signalfx_single_value_chart":       singleValueChartResource(),
			"signalfx_team":                     teamResource(),
			"signalfx_time_chart":               timeChartResource(),
			"signalfx_text_chart":               textChartResource(),
			"signalfx_victor_ops_integration":   integrationVictorOpsResource(),
			"signalfx_webhook_integration":      integrationWebhookResource(),
		},
		ConfigureFunc: signalfxConfigure,
	}

	return sfxProvider
}

func signalfxConfigure(data *schema.ResourceData) (interface{}, error) {
	config := signalfxConfig{}

	// /etc/signalfx.conf has lowest priority
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
		HomeConfigPath = usr.HomeDir + HomeConfigSuffix
		if err != nil {
			return nil, fmt.Errorf("Failed to get user environment %s", err.Error())
		}
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

	if config.AuthToken == "" {
		return &config, fmt.Errorf("auth_token: required field is not set")
	}
	if url, ok := data.GetOk("api_url"); ok {
		config.APIURL = url.(string)
	}
	if custom_app_url, ok := data.GetOk("custom_app_url"); ok {
		config.CustomAppURL = custom_app_url.(string)
	}

	var netTransport = logging.NewTransport("SignalFx", &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	})

	pv := version.ProviderVersion
	providerUserAgent := fmt.Sprintf("Terraform/%s terraform-provider-signalfx/%s", sfxProvider.TerraformVersion, pv)

	totalTimeoutSeconds := data.Get("timeout_seconds").(int)
	log.Printf("[DEBUG] SignalFx: HTTP Timeout is %d seconds", totalTimeoutSeconds)
	client, err := sfx.NewClient(config.AuthToken,
		sfx.APIUrl(config.APIURL),
		sfx.HTTPClient(&http.Client{
			Timeout:   time.Second * time.Duration(int64(totalTimeoutSeconds)),
			Transport: netTransport,
		}),
		sfx.UserAgent(fmt.Sprintf(providerUserAgent)),
	)
	if err != nil {
		return &config, err
	}

	config.Client = client

	return &config, nil
}

func readConfigFile(configPath string, config *signalfxConfig) error {
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to open config file. %s", err.Error())
	}
	err = json.Unmarshal(configFile, config)
	if err != nil {
		return fmt.Errorf("Failed to parse config file. %s", err.Error())
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
	net, err := netrc.ParseFile(path)
	if err != nil {
		return fmt.Errorf("Error parsing netrc file at %q: %s", path, err)
	}

	machine := net.FindMachine("api.signalfx.com")
	if machine == nil {
		// Machine not found, no problem
		return nil
	}

	// Set the auth token
	config.AuthToken = machine.Password
	return nil
}
