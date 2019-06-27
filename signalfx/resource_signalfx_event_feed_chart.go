package signalfx

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
)

func eventFeedChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the chart",
			},
			"program_text": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Signalflow program text for the chart. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the chart (Optional)",
			},
			"viz_options": &schema.Schema{
				Type:        schema.TypeSet,
				Deprecated:  "signalfx_event_feed_chart.viz_options is being removed in the next release",
				Optional:    true,
				Description: "Plot-level customization options, associated with a publish statement",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label used in the publish statement that displays the plot (metric time series data) you want to customize",
						},
						"color": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Color to use",
							ValidateFunc: validatePerSignalColor,
						},
					},
				},
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: eventFeedChartCreate,
		Read:   eventFeedChartRead,
		Update: eventFeedChartUpdate,
		Delete: eventFeedChartDelete,
	}
}

/*
  Use Resource object to construct json payload in order to create an event feed chart
*/
func getPayloadEventFeedChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	return &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Options: &chart.Options{
			Type: "Event",
		},
	}
}

func eventFeedChartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadEventFeedChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Event Feed Chart Payload: %s", string(debugOutput))

	c, err := config.Client.CreateChart(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return eventfeedchartAPIToTF(d, c)
}

func eventfeedchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	log.Printf("[DEBUG] SignalFx: Got Event Feed Chart to enState %v", c)

	if err := d.Set("name", c.Name); err != nil {
		return err
	}
	if err := d.Set("description", c.Description); err != nil {
		return err
	}
	if err := d.Set("program_text", c.ProgramText); err != nil {
		return err
	}

	return nil
}

func eventFeedChartRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	c, err := config.Client.GetChart(d.Id())
	if err != nil {
		return err
	}

	return eventfeedchartAPIToTF(d, c)
}

func eventFeedChartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadEventFeedChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Event Feed Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Event Feed Chart Response: %v", c)

	d.SetId(c.Id)
	return eventfeedchartAPIToTF(d, c)
}

func eventFeedChartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(d.Id())
}
