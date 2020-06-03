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

func integrationOpsgenieResource() *schema.Resource {
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
				Required:    true,
				Description: "Opsgenie API key",
				Sensitive:   true,
			},
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Opsgenie API URL for integration",
			},
		},

		Create: integrationOpsgenieCreate,
		Read:   integrationOpsgenieRead,
		Update: integrationOpsgenieUpdate,
		Delete: integrationOpsgenieDelete,
		Exists: integrationOpsgenieExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationOpsgenieExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetOpsgenieIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getOpsgeniePayloadIntegration(d *schema.ResourceData) *integration.OpsgenieIntegration {
	return &integration.OpsgenieIntegration{
		Type:    "Opsgenie",
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		ApiKey:  d.Get("api_key").(string),
		ApiUrl:  d.Get("api_url").(string),
	}
}

func integrationOpsgenieRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetOpsgenieIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return opsgenieIntegrationAPIToTF(d, int)
}

func opsgenieIntegrationAPIToTF(d *schema.ResourceData, og *integration.OpsgenieIntegration) error {
	debugOutput, _ := json.Marshal(og)
	log.Printf("[DEBUG] SignalFx: Got Opsgenie Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", og.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", og.Enabled); err != nil {
		return err
	}
	// Note, the API doesn't return a Webhook URL so we ignore it
	return nil
}

func integrationOpsgenieCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getOpsgeniePayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Opsgenie Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateOpsgenieIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return opsgenieIntegrationAPIToTF(d, int)
}

func integrationOpsgenieUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getOpsgeniePayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Opsgenie Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateOpsgenieIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return opsgenieIntegrationAPIToTF(d, int)
}

func integrationOpsgenieDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteOpsgenieIntegration(context.TODO(), d.Id())
}
