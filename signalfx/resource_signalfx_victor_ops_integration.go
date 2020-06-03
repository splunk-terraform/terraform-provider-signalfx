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

func integrationVictorOpsResource() *schema.Resource {
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
			"post_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Opsgenie API URL for integration",
			},
		},

		Create: integrationVictorOpsCreate,
		Read:   integrationVictorOpsRead,
		Update: integrationVictorOpsUpdate,
		Delete: integrationVictorOpsDelete,
		Exists: integrationVictorOpsExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationVictorOpsExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetVictorOpsIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getVictorOpsPayloadIntegration(d *schema.ResourceData) *integration.VictorOpsIntegration {
	return &integration.VictorOpsIntegration{
		Type:    "VictorOps",
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		PostUrl: d.Get("post_url").(string),
	}
}

func integrationVictorOpsRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetVictorOpsIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return victorOpsIntegrationAPIToTF(d, int)
}

func victorOpsIntegrationAPIToTF(d *schema.ResourceData, og *integration.VictorOpsIntegration) error {
	debugOutput, _ := json.Marshal(og)
	log.Printf("[DEBUG] SignalFx: Got VictorOps Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", og.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", og.Enabled); err != nil {
		return err
	}
	// Note, the API doesn't return a POST URL so we ignore it
	return nil
}

func integrationVictorOpsCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getVictorOpsPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create VictorOps Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateVictorOpsIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return victorOpsIntegrationAPIToTF(d, int)
}

func integrationVictorOpsUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getVictorOpsPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update VictorOps Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateVictorOpsIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return victorOpsIntegrationAPIToTF(d, int)
}

func integrationVictorOpsDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteVictorOpsIntegration(context.TODO(), d.Id())
}
