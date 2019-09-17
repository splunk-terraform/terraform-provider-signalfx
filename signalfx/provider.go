package signalfx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	homedir "github.com/mitchellh/go-homedir"
	sfx "github.com/signalfx/signalfx-go"
)

var SystemConfigPath = "/etc/signalfx.conf"
var HomeConfigSuffix = "/.signalfx.conf"
var HomeConfigPath = ""

type signalfxConfig struct {
	AuthToken    string `json:"auth_token"`
	APIURL       string `json:"api_url"`
	CustomAppURL string `json:"custom_app_url"`
	Client       *sfx.Client
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
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
				Default:     "https://api.signalfx.com",
				Description: "API URL for your SignalFx org, may include a realm",
			},
			"custom_app_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://app.signalfx.com",
				Description: "Application URL for your SignalFx org, often customzied for organizations using SSO",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"signalfx_aws_external_integration": integrationAWSExternalResource(),
			// "signalfx_aws_token_integration": integrationAWSTokenResource(),
			"signalfx_aws_integration":        integrationAWSResource(),
			"signalfx_azure_integration":      integrationAzureResource(),
			"signalfx_dashboard":              dashboardResource(),
			"signalfx_dashboard_group":        dashboardGroupResource(),
			"signalfx_detector":               detectorResource(),
			"signalfx_event_feed_chart":       eventFeedChartResource(),
			"signalfx_gcp_integration":        integrationGCPResource(),
			"signalfx_heatmap_chart":          heatmapChartResource(),
			"signalfx_list_chart":             listChartResource(),
			"signalfx_org_token":              orgTokenResource(),
			"signalfx_opsgenie_integration":   integrationOpsgenieResource(),
			"signalfx_pagerduty_integration":  integrationPagerDutyResource(),
			"signalfx_slack_integration":      integrationSlackResource(),
			"signalfx_single_value_chart":     singleValueChartResource(),
			"signalfx_time_chart":             timeChartResource(),
			"signalfx_text_chart":             textChartResource(),
			"signalfx_victor_ops_integration": integrationVictorOpsResource(),
		},
		ConfigureFunc: signalfxConfigure,
	}
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

	client, err := sfx.NewClient(config.AuthToken, sfx.APIUrl(config.APIURL))
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
