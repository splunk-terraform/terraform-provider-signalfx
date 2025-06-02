// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

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
		},

		Create: integrationPagerDutyCreate,
		Read:   integrationPagerDutyRead,
		Update: integrationPagerDutyUpdate,
		Delete: integrationPagerDutyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationPagerDutyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetPagerDutyIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return pagerDutyIntegrationAPIToTF(d, int)
}

func pagerDutyIntegrationAPIToTF(d *schema.ResourceData, pd *integration.PagerDutyIntegration) error {
	debugOutput, _ := json.Marshal(pd)
	log.Printf("[DEBUG] SignalFx: Got PagerDuty Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", pd.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", pd.Enabled); err != nil {
		return err
	}
	// Note, the API doesn't return API keys, so we ignore that

	return nil
}

func getPayloadPagerDutyIntegration(d *schema.ResourceData) (*integration.PagerDutyIntegration, error) {
	return &integration.PagerDutyIntegration{
		Type:    "PagerDuty",
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		ApiKey:  d.Get("api_key").(string),
	}, nil
}

func integrationPagerDutyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	payload, err := getPayloadPagerDutyIntegration(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create PagerDuty Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreatePagerDutyIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)
	return pagerDutyIntegrationAPIToTF(d, int)
}

func integrationPagerDutyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	payload, err := getPayloadPagerDutyIntegration(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update PagerDuty Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdatePagerDutyIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}

	return pagerDutyIntegrationAPIToTF(d, int)
}

func integrationPagerDutyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeletePagerDutyIntegration(context.TODO(), d.Id())
}
