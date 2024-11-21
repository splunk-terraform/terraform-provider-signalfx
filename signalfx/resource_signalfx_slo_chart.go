// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
	"log"
)

func sloChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"slo_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the attached SLO",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: slochartCreate,
		Read:   slochartRead,
		Update: slochartUpdate,
		Delete: slochartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

/*
Use Resource object to construct json payload in order to create an SLO chart
*/
func getPayloadSloChart(d *schema.ResourceData) *chart.CreateUpdateSloChartRequest {
	return &chart.CreateUpdateSloChartRequest{
		SloId: d.Get("slo_id").(string),
	}
}

func slochartCreate(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadSloChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create SLO Chart Payload: %s", string(debugOutput))

	sloChart, err := config.Client.CreateSloChart(context.TODO(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+sloChart.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(sloChart.Id)
	return slochartAPIToTF(d, sloChart)
}

func slochartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got SLO Chart to enState: %s", string(debugOutput))

	if err := d.Set("slo_id", c.SloId); err != nil {
		return err
	}

	return nil
}

func slochartRead(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)
	c, err := config.Client.GetChart(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}

	return slochartAPIToTF(d, c)
}

func slochartUpdate(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadSloChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update SLO Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateSloChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update SLO Chart Response: %v", c)

	d.SetId(c.Id)
	return slochartAPIToTF(d, c)
}

func slochartDelete(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
