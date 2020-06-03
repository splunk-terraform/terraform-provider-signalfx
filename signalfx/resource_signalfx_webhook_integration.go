package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationWebhookResource() *schema.Resource {
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
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Webhook URL",
			},
			"shared_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
				Sensitive:   true,
			},
			"headers": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "HTTP headers to pass in the request",
				Sensitive:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"header_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"header_value": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},
		},

		Create: integrationWebhookCreate,
		Read:   integrationWebhookRead,
		Update: integrationWebhookUpdate,
		Delete: integrationWebhookDelete,
		Exists: integrationWebhookExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationWebhookExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetWebhookIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getWebhookPayloadIntegration(d *schema.ResourceData) *integration.WebhookIntegration {
	webhook := &integration.WebhookIntegration{
		Type:    "Webhook",
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		Url:     d.Get("url").(string),
	}

	if val, ok := d.GetOk("shared_secret"); ok {
		webhook.SharedSecret = val.(string)
	}

	if val, ok := d.GetOk("headers"); ok {
		hs := val.(*schema.Set).List()
		headers := make(map[string]interface{})
		for _, v := range hs {
			v := v.(map[string]interface{})
			headers[v["header_key"].(string)] = v["header_value"].(string)
		}
		webhook.Headers = headers
	}

	return webhook
}

func integrationWebhookRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetWebhookIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return webhookIntegrationAPIToTF(d, int)
}

func webhookIntegrationAPIToTF(d *schema.ResourceData, og *integration.WebhookIntegration) error {
	debugOutput, _ := json.Marshal(og)
	log.Printf("[DEBUG] SignalFx: Got Webhook Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", og.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", og.Enabled); err != nil {
		return err
	}
	if err := d.Set("url", og.Url); err != nil {
		return err
	}
	if err := d.Set("shared_secret", og.SharedSecret); err != nil {
		return err
	}
	if len(og.Headers) > 0 {
		headers := make([]map[string]interface{}, len(og.Headers))
		count := 0
		for k, v := range og.Headers {
			headers[count] = map[string]interface{}{
				"header_key":   k,
				"header_value": v,
			}
			count++
		}
		if err := d.Set("headers", headers); err != nil {
			return err
		}
	}
	return nil
}

func integrationWebhookCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getWebhookPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Webhook Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateWebhookIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return webhookIntegrationAPIToTF(d, int)
}

func integrationWebhookUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getWebhookPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Webhook Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateWebhookIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return webhookIntegrationAPIToTF(d, int)
}

func integrationWebhookDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteWebhookIntegration(context.TODO(), d.Id())
}
