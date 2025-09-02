// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sfxgo "github.com/signalfx/signalfx-go"
	chart "github.com/signalfx/signalfx-go/chart"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func textChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the chart",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the chart (Optional)",
			},
			"markdown": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Markdown text to display. More info at: https://github.com/adam-p/markdown-here/wiki/Markdown-Cheatsheet",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
			"tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the resource",
			},
		},

		Create: textchartCreate,
		Read:   textchartRead,
		Update: textchartUpdate,
		Delete: textchartDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
Use Resource object to construct json payload in order to create a text chart
*/
func getPayloadTextChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	return &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Options: &chart.Options{
			Type:     "Text",
			Markdown: d.Get("markdown").(string),
		},
		Tags: convert.SchemaListAll(d.Get("tags"), convert.ToString),
	}
}

func textchartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadTextChart(d)

	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Text Chart Payload: %s", string(debugOutput))

	c, err := config.Client.CreateChart(context.TODO(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return textchartAPIToTF(d, c)
}

func textchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Text Chart to enState: %s", string(debugOutput))

	if err := d.Set("name", c.Name); err != nil {
		return err
	}
	if err := d.Set("description", c.Description); err != nil {
		return err
	}
	if err := d.Set("markdown", c.Options.Markdown); err != nil {
		return err
	}

	return nil
}

func isTextchartNotFound(err error) bool {
	sfxRespErr, ok := err.(*sfxgo.ResponseError)
	return ok && sfxRespErr.Code() == http.StatusNotFound
}

func textchartRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	c, err := config.Client.GetChart(context.TODO(), d.Id())
	if err != nil {
		if isTextchartNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}

	return textchartAPIToTF(d, c)
}

func textchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadTextChart(d)

	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Text Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Text Chart Response: %v", c)

	d.SetId(c.Id)
	return textchartAPIToTF(d, c)
}

func textchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
