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

func integrationSlackResource() *schema.Resource {
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
			"webhook_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Slack Webhook URL for integration",
				Sensitive:   true,
			},
		},

		Create: integrationSlackCreate,
		Read:   integrationSlackRead,
		Update: integrationSlackUpdate,
		Delete: integrationSlackDelete,
		Exists: integrationSlackExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationSlackExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetSlackIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getSlackPayloadIntegration(d *schema.ResourceData) *integration.SlackIntegration {
	return &integration.SlackIntegration{
		Type:       "Slack",
		Name:       d.Get("name").(string),
		Enabled:    d.Get("enabled").(bool),
		WebhookUrl: d.Get("webhook_url").(string),
	}
}

func integrationSlackRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetSlackIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return slackIntegrationAPIToTF(d, int)
}

func slackIntegrationAPIToTF(d *schema.ResourceData, slack *integration.SlackIntegration) error {
	debugOutput, _ := json.Marshal(slack)
	log.Printf("[DEBUG] SignalFx: Got Slack Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", slack.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", slack.Enabled); err != nil {
		return err
	}
	// Note, the API doesn't return a Webhook URL so we ignore it
	return nil
}

func integrationSlackCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getSlackPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Slack Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateSlackIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return slackIntegrationAPIToTF(d, int)
}

func integrationSlackUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getSlackPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Slack Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateSlackIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return slackIntegrationAPIToTF(d, int)
}

func integrationSlackDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteSlackIntegration(context.TODO(), d.Id())
}
