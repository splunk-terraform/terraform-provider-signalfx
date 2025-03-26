// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	chart "github.com/signalfx/signalfx-go/chart"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func tableChartResource() *schema.Resource {
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
				ValidateFunc: validation.StringLenBetween(18, 50000),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the chart (Optional)",
			},
			"unit_prefix": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Metric",
				Description: "(Metric by default) Must be \"Metric\" or \"Binary\"",
				ValidateFunc: validation.StringInSlice([]string{
					"Metric", "Binary",
				}, false),
			},
			"minimum_resolution": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The minimum resolution (in seconds) to use for computing the underlying program",
			},
			"max_delay": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 900),
				Description:  "How long (in seconds) to wait for late datapoints",
			},
			"timezone": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UTC",
				Description: "The property value is a string that denotes the geographic region associated with the time zone, (e.g. Australia/Sydney)",
			},
			"refresh_interval": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "How often (in seconds) to refresh the values of the Table",
			},
			"disable_sampling": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) If false, samples a subset of the output MTS, which improves UI performance",
			},
			"group_by": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Properties to group by in the Table (in nesting order)",
			},
			"hide_timestamp": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) Whether to show the timestamp in the chart",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
			"viz_options": &schema.Schema{
				Type:        schema.TypeSet,
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
						"display_name": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.",
						},
						"value_unit": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateUnitTimeChart,
							Description:  "A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes)",
						},
						"value_prefix": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An arbitrary prefix to display with the value of this plot",
						},
						"value_suffix": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An arbitrary suffix to display with the value of this plot",
						},
					},
				},
			},
			"tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the resource",
			},
		},

		Create: tablechartCreate,
		Read:   tablechartRead,
		Update: tablechartUpdate,
		Delete: tablechartDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
Use Resource object to construct json payload in order to create an Table chart
*/
func getPayloadTableChart(d *schema.ResourceData) (*chart.CreateUpdateChartRequest, error) {
	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Tags:        convert.SchemaListAll(d.Get("tags"), convert.ToString),
	}

	options, err := getTableOptionsChart(d)
	if err != nil {
		return nil, err
	}

	if vizOptions := getPerSignalVizOptions(d, false); len(vizOptions) > 0 {
		options.PublishLabelOptions = vizOptions
	}

	payload.Options = options

	return payload, nil
}

func getTableOptionsChart(d *schema.ResourceData) (*chart.Options, error) {
	options := &chart.Options{
		Type: "TableChart",
	}
	if val, ok := d.GetOk("unit_prefix"); ok {
		options.UnitPrefix = val.(string)
	}
	if refreshInterval, ok := d.GetOk("refresh_interval"); ok {
		ri := int32(refreshInterval.(int) * 1000)
		options.RefreshInterval = &ri
	}
	if timestampHidden, ok := d.GetOk("hide_timestamp"); ok {
		options.TimestampHidden = timestampHidden.(bool)
	}

	var groupBy []string
	if val, ok := d.GetOk("group_by"); ok {
		groupBy = []string{}
		for _, g := range val.([]interface{}) {
			groupBy = append(groupBy, g.(string))
		}
	}
	options.GroupBy = groupBy

	var programOptions *chart.GeneralOptions
	if val, ok := d.GetOk("minimum_resolution"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		mr := int32(val.(int) * 1000)
		programOptions.MinimumResolution = &mr
	}
	if val, ok := d.GetOk("max_delay"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		md := int32(val.(int) * 1000)
		programOptions.MaxDelay = &md
	}
	if val, ok := d.GetOk("timezone"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.Timezone = val.(string)
	}
	if val, ok := d.GetOk("disable_sampling"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.DisableSampling = val.(bool)
	}
	options.ProgramOptions = programOptions

	return options, nil
}

func tablechartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadTableChart(d)
	if err != nil {
		return err
	}
	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Table Chart Payload: %s", string(debugOutput))

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

	return tablechartAPIToTF(d, c)
}

func tablechartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Table Chart to enState: %s", string(debugOutput))

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
	if err := d.Set("unit_prefix", options.UnitPrefix); err != nil {
		return err
	}
	if options.RefreshInterval != nil {
		if err := d.Set("refresh_interval", *options.RefreshInterval/1000); err != nil {
			return err
		}
	}
	if err := d.Set("group_by", options.GroupBy); err != nil {
		return err
	}
	if err := d.Set("hide_timestamp", options.TimestampHidden); err != nil {
		return err
	}
	if options.ColorRange != nil && options.ColorRange.Color != "" {
		colorRange := make([]map[string]interface{}, 1)
		colorRange[0] = map[string]interface{}{
			"min_value": options.ColorRange.Min,
			"max_value": options.ColorRange.Max,
			"color":     options.ColorRange.Color,
		}
		if err := d.Set("color_range", colorRange); err != nil {
			return err
		}
	} else if options.ColorScale2 != nil {
		colorScale, err := decodeColorScale(options)
		if err != nil {
			return err
		}
		if err := d.Set("color_scale", colorScale); err != nil {
			return err
		}
	}

	if options.ProgramOptions != nil {
		if options.ProgramOptions.MinimumResolution != nil {
			if err := d.Set("minimum_resolution", *options.ProgramOptions.MinimumResolution/1000); err != nil {
				return err
			}
		}
		if options.ProgramOptions.MaxDelay != nil {
			if err := d.Set("max_delay", *options.ProgramOptions.MaxDelay/1000); err != nil {
				return err
			}
		}
		if err := d.Set("timezone", options.ProgramOptions.Timezone); err != nil {
			return err
		}
		if err := d.Set("disable_sampling", options.ProgramOptions.DisableSampling); err != nil {
			return err
		}
	}

	if len(options.PublishLabelOptions) > 0 {
		plos := make([]map[string]interface{}, len(options.PublishLabelOptions))
		for i, plo := range options.PublishLabelOptions {
			no, err := publishNonTimeLabelOptionsToMap(plo)
			if err != nil {
				return err
			}
			plos[i] = no
		}
		if err := d.Set("viz_options", plos); err != nil {
			return err
		}
	}

	return nil
}

func tablechartRead(d *schema.ResourceData, meta interface{}) error {
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

	return tablechartAPIToTF(d, c)
}

func tablechartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadTableChart(d)
	if err != nil {
		return err
	}

	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Table Chart Response: %v", c)

	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return tablechartAPIToTF(d, c)
}

func tablechartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
