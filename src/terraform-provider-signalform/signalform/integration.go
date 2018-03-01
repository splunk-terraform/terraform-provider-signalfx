package signalform

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

const (
	INTEGRATION_API_URL = "https://api.signalfx.com/v2/integration"
)

func integrationResource() *schema.Resource {
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
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Type of the integration",
				ValidateFunc: validateIntegrationType,
			},
			"api_key": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "PagerDuty API key",
				Sensitive:     true,
				ConflictsWith: []string{"webhook_url"},
			},
			"webhook_url": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Slack Incoming Webhook URL",
				Sensitive:     true,
				ConflictsWith: []string{"api_key"},
			},
		},

		Create: integrationCreate,
		Read:   integrationRead,
		Update: integrationUpdate,
		Delete: integrationDelete,
	}
}

func validateIntegrationType(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"PagerDuty", "Slack"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}

func getPayloadIntegration(d *schema.ResourceData) ([]byte, error) {
	integrationType := d.Get("type").(string)
	payload := map[string]interface{}{
		"name":    d.Get("name").(string),
		"enabled": d.Get("enabled").(bool),
		"type":    integrationType,
	}

	switch integrationType {
	case "PagerDuty":
		payload["apiKey"] = d.Get("api_key").(string)
	case "Slack":
		payload["webhookUrl"] = d.Get("webhook_url").(string)
	}

	return json.Marshal(payload)
}

func integrationCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	payload, err := getPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	return resourceCreate(INTEGRATION_API_URL, config.AuthToken, payload, d)
}

func integrationRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	url := fmt.Sprintf("%s/%s", INTEGRATION_API_URL, d.Id())

	return resourceRead(url, config.AuthToken, d)
}

func integrationUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	payload, err := getPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	url := fmt.Sprintf("%s/%s", INTEGRATION_API_URL, d.Id())

	return resourceUpdate(url, config.AuthToken, payload, d)
}

func integrationDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	url := fmt.Sprintf("%s/%s", INTEGRATION_API_URL, d.Id())
	return resourceDelete(url, config.AuthToken, d)
}
