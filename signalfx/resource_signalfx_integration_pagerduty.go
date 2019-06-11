package signalfx

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// This resource leverages common methods for read and delete from
// integration.go!

func integrationPagerDutyResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "PagerDuty API key",
				Sensitive:   true,
			},
			"synced": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing.",
			},
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
			},
		},

		Create: integrationPagerDutyCreate,
		Read:   integrationRead,
		Update: integrationPagerDutyUpdate,
		Delete: integrationDelete,
	}
}

func getPagerDutyPayloadIntegration(d *schema.ResourceData) ([]byte, error) {
	payload := map[string]interface{}{
		"name":    d.Get("name").(string),
		"enabled": d.Get("enabled").(bool),
		"type":    "PagerDuty",
		"apiKey":  d.Get("api_key").(string),
	}
	return json.Marshal(payload)
}

func integrationPagerDutyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPagerDutyPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	url, err := buildURL(config.APIURL, INTEGRATION_API_PATH, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceCreate(url, config.AuthToken, payload, d)
}

func integrationPagerDutyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPagerDutyPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	path := fmt.Sprintf("%s/%s", INTEGRATION_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceUpdate(url, config.AuthToken, payload, d)
}
