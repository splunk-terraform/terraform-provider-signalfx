package signalfx

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

const (
	INTEGRATION_API_PATH = "/v2/integration"
)

func integrationResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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
				ConflictsWith: []string{"webhook_url", "poll_rate", "services", "project_service_keys"},
			},
			"webhook_url": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Slack Incoming Webhook URL",
				Sensitive:     true,
				ConflictsWith: []string{"api_key", "poll_rate", "services", "project_service_keys"},
			},
			"poll_rate": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "GCP poll rate",
				ConflictsWith: []string{"api_key", "webhook_url"},
				ValidateFunc:  validatePollRate,
			},
			"services": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "GCP enabled services",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"api_key", "webhook_url"},
			},
			"project_service_keys": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "GCP project service keys",
				Sensitive:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"project_key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"api_key", "webhook_url"},
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
	allowedWords := []string{"PagerDuty", "Slack", "GCP"}
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
	case "GCP":
		payload["pollRate"] = d.Get("poll_rate").(int)
		payload["services"] = expandServices(d.Get("services").([]interface{}))
		payload["projectServiceKeys"] = expandProjectServiceKeys(d.Get("project_service_keys").([]interface{}))
	}

	return json.Marshal(payload)
}

func integrationCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	url, err := buildURL(config.APIURL, INTEGRATION_API_PATH)
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceCreate(url, config.AuthToken, payload, d)
}

func integrationRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", INTEGRATION_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path)
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceRead(url, config.AuthToken, d)
}

func integrationUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	path := fmt.Sprintf("%s/%s", INTEGRATION_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path)
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceUpdate(url, config.AuthToken, payload, d)
}

func integrationDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", INTEGRATION_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path)
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceDelete(url, config.AuthToken, d)
}

func expandServices(services []interface{}) []string {
	if len(services) == 0 {
		return []string{}
	}
	payload := make([]string, 0, len(services))
	for _, service := range services {
		if service != nil {
			payload = append(payload, service.(string))
		}
	}
	return payload
}

func expandProjectServiceKeys(projects []interface{}) []map[string]string {
	if len(projects) == 0 {
		return []map[string]string{}
	}
	payload := make([]map[string]string, 0, len(projects))
	for _, project := range projects {
		m := project.(map[string]interface{})
		config := map[string]string{
			"projectId":  m["project_id"].(string),
			"projectKey": m["project_key"].(string),
		}
		payload = append(payload, config)
	}
	return payload
}

func validatePollRate(value interface{}, key string) (warns []string, errors []error) {
	v := value.(int)
	if v != 60000 && v != 300000 {
		errors = append(errors, fmt.Errorf("%q must be either 60000 or 300000, got: %d", key, v))
	}
	return
}
