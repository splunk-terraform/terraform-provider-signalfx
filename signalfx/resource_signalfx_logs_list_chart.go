package signalfx

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
)

func logsListChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the chart",
			},
			"program_text": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Signalflow program text for the chart. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
				ValidateFunc: validation.StringLenBetween(16, 50000),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the chart (Optional)",
			},
			"time_range": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds to display in the visualization. This is a rolling range from the current time. Example: 3600 = `-1h`",
				ConflictsWith: []string{"start_time", "end_time"},
			},
			"start_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds since epoch to start the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"end_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds since epoch to end the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"columns": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Column configuration",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the column",
						},
					},
				},
			},
			"sort_options": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Sorting options configuration",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the column",
						},
						"descending": &schema.Schema{
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Name of the column",
						},
					},
				},
			},
			"default_connection": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "default connection that the dashboard uses",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: logslistchartCreate,
		Read:   logslistchartRead,
		Update: logslistchartUpdate,
		Delete: logslistchartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a logs list  chart
*/
func getPayloadLogsListChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	var timeOptions *chart.TimeDisplayOptions
	if val, ok := d.GetOk("time_range"); ok {
		r := int64(val.(int) * 1000)
		timeOptions = &chart.TimeDisplayOptions{
			Range: &r,
			Type:  "relative",
		}
	}
	if val, ok := d.GetOk("start_time"); ok {
		start := int64(val.(int) * 1000)
		timeOptions = &chart.TimeDisplayOptions{
			Start: &start,
			Type:  "absolute",
		}
		if val, ok := d.GetOk("end_time"); ok {
			end := int64(val.(int) * 1000)
			timeOptions.End = &end
		}
	}
	return &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Options: &chart.Options{
			Time: timeOptions,
			Type: "LogsChart",
		},
	}
}

func logslistchartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadLogsListChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Logs List Chart Payload: %s", string(debugOutput))

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
	log.Printf("[DEBUG] appURL in create: %s", string(appURL))

	return logslistchartAPIToTF(d, c)
}

func logslistchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Logs List Chart to enState: %s", string(debugOutput))

	if err := d.Set("name", c.Name); err != nil {
		return err
	}
	if err := d.Set("description", c.Description); err != nil {
		return err
	}
	if err := d.Set("program_text", c.ProgramText); err != nil {
		return err
	}

	options := c.Options

	if options.Time != nil {
		if options.Time.Type == "relative" {
			if options.Time.Range != nil {
				if err := d.Set("time_range", *options.Time.Range/1000); err != nil {
					return err
				}
			}
		} else {
			if options.Time.Start != nil {
				if err := d.Set("start_time", *options.Time.Start/1000); err != nil {
					return err
				}
			}
			if options.Time.End != nil {
				if err := d.Set("end_time", *options.Time.End/1000); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func logslistchartRead(d *schema.ResourceData, meta interface{}) error {
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
	log.Printf("[DEBUG] appURL in read: %s", string(appURL))

	return logslistchartAPIToTF(d, c)
}

func logslistchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadLogsListChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Logs List Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Logs List Chart Response: %v", c)

	d.SetId(c.Id)
	return logslistchartAPIToTF(d, c)
}

func logslistchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
