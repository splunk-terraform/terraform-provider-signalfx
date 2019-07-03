package signalfx

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// This resource leverages common methods for read and delete from
// integration.go!

func integrationSlackResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
			},
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
			"webhook_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Slack Webhook URL for integration",
				Sensitive:   true,
			},
			"synced": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing.",
			},
		},

		Create: integrationSlackCreate,
		Read:   integrationRead,
		Update: integrationSlackUpdate,
		Delete: integrationDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getSlackPayloadIntegration(d *schema.ResourceData) ([]byte, error) {
	payload := map[string]interface{}{
		"name":       d.Get("name").(string),
		"enabled":    d.Get("enabled").(bool),
		"type":       "Slack",
		"webhookUrl": d.Get("webhook_url").(string),
	}
	return json.Marshal(payload)
}

func integrationSlackCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getSlackPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	url, err := buildURL(config.APIURL, INTEGRATION_API_PATH, map[string]string{})
	if err != nil {
		return fmt.Errorf("[DEBUG] SignalFx: Error constructing API URL: %s", err.Error())
	}

	return resourceCreate(url, config.AuthToken, payload, d)
}

func integrationSlackUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getSlackPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	path := fmt.Sprintf("%s/%s", INTEGRATION_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[DEBUG] SignalFx: Error constructing API URL: %s", err.Error())
	}

	return resourceUpdate(url, config.AuthToken, payload, d)
}
