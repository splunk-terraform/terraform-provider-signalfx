package signalfx

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	chart "github.com/signalfx/signalfx-go/chart"
)

func singleValueChartResource() *schema.Resource {
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
			"color_by": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Metric",
				Description: "(Metric by default) Must be \"Metric\", \"Dimension\", or \"Scale\". \"Scale\" maps to Color by Value in the UI",
				ValidateFunc: validation.StringInSlice([]string{
					"Metric", "Dimension", "Scale",
				}, false),
			},
			"max_delay": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "How long (in seconds) to wait for late datapoints",
				ValidateFunc: validation.IntBetween(0, 900),
			},
			"refresh_interval": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "How often (in seconds) to refresh the values of the list",
			},
			"max_precision": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum precision to for values displayed in the list",
			},
			"is_timestamp_hidden": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) Whether to hide the timestamp in the chart",
			},
			"show_spark_line": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "(false by default) Whether to show a trend line below the current value",
				Default:     false,
			},
			"secondary_visualization": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "None",
				Description:  "(false by default) What kind of secondary visualization to show (None, Radial, Linear, Sparkline)",
				ValidateFunc: validateSecondaryVisualization,
			},
			"color_scale": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Single color range including both the color to display for that range and the borders of the range",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"color": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The color to use. Must be either \"gray\", \"blue\", \"navy\", \"orange\", \"yellow\", \"magenta\", \"purple\", \"violet\", \"lilac\", \"green\", \"aquamarine\"",
							ValidateFunc: validateHeatmapChartColor,
						},
						"gt": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat32,
							Description: "Indicates the lower threshold non-inclusive value for this range",
						},
						"gte": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat32,
							Description: "Indicates the lower threshold inclusive value for this range",
						},
						"lt": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat32,
							Description: "Indicates the upper threshold non-inculsive value for this range",
						},
						"lte": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat32,
							Description: "Indicates the upper threshold inclusive value for this range",
						},
					},
				},
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
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: singlevaluechartCreate,
		Read:   singlevaluechartRead,
		Update: singlevaluechartUpdate,
		Delete: singlevaluechartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a single value chart
*/
func getPayloadSingleValueChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
	}

	viz := getSingleValueChartOptions(d)
	if vizOptions := getPerSignalVizOptions(d); len(vizOptions) > 0 {
		viz.PublishLabelOptions = vizOptions
	}
	payload.Options = viz

	return payload
}

func getSingleValueChartOptions(d *schema.ResourceData) *chart.Options {
	options := &chart.Options{
		Type: "SingleValue",
	}
	if val, ok := d.GetOk("unit_prefix"); ok {
		options.UnitPrefix = val.(string)
	}
	if val, ok := d.GetOk("color_by"); ok {
		cb := val.(string)
		options.ColorBy = cb
		if cb == "Scale" {
			if colorScaleOptions := getColorScaleOptions(d); len(colorScaleOptions) > 0 {
				options.ColorScale2 = colorScaleOptions
			}

		}
	}

	var programOptions *chart.GeneralOptions
	if val, ok := d.GetOk("max_delay"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		md := int32(val.(int) * 1000)
		programOptions.MaxDelay = &md
	}
	options.ProgramOptions = programOptions

	if refreshInterval, ok := d.GetOk("refresh_interval"); ok {
		ri := int32(refreshInterval.(int) * 1000)
		options.RefreshInterval = &ri
	}
	if maxPrecision, ok := d.GetOk("max_precision"); ok {
		mp := int32(maxPrecision.(int))
		options.MaximumPrecision = &mp
	}

	if val, ok := d.GetOk("secondary_visualization"); ok {
		secondaryVisualization := val.(string)
		if secondaryVisualization != "" {
			options.SecondaryVisualization = secondaryVisualization
		}
	}
	if isTimestampHidden, ok := d.GetOk("is_timestamp_hidden"); ok {
		options.TimestampHidden = isTimestampHidden.(bool)
	}
	if showSparkLine, ok := d.GetOk("show_spark_line"); ok {
		options.ShowSparkLine = showSparkLine.(bool)
	}

	return options
}

func singlevaluechartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadSingleValueChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Single Value Chart Payload: %s", string(debugOutput))

	chart, err := config.Client.CreateChart(context.TODO(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+chart.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(chart.Id)
	return singlevaluechartAPIToTF(d, chart)
}

func singlevaluechartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Single Value Chart to enState: %s", string(debugOutput))

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
	if err := d.Set("color_by", options.ColorBy); err != nil {
		return err
	}
	if options.RefreshInterval != nil {
		if err := d.Set("refresh_interval", *options.RefreshInterval/1000); err != nil {
			return err
		}
	}
	if err := d.Set("max_precision", options.MaximumPrecision); err != nil {
		return err
	}
	if err := d.Set("secondary_visualization", options.SecondaryVisualization); err != nil {
		return err
	}
	if err := d.Set("is_timestamp_hidden", options.TimestampHidden); err != nil {
		return err
	}
	if err := d.Set("show_spark_line", options.ShowSparkLine); err != nil {
		return err
	}
	if options.ProgramOptions != nil {
		if options.ProgramOptions.MaxDelay != nil {
			if err := d.Set("max_delay", *options.ProgramOptions.MaxDelay/1000); err != nil {
				return err
			}
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

	if options.ColorBy == "Scale" && len(options.ColorScale2) > 0 {
		colorScale, err := decodeColorScale(options)
		if err != nil {
			return err
		}
		if err := d.Set("color_scale", colorScale); err != nil {
			return err
		}
	}

	return nil
}

func decodeColorScale(options *chart.Options) ([]map[string]interface{}, error) {
	scales := make([]map[string]interface{}, len(options.ColorScale2))
	for i, cs := range options.ColorScale2 {
		scale := map[string]interface{}{}
		if cs.Gt == nil {
			scale["gt"] = math.MaxFloat32
		} else {
			scale["gt"] = *cs.Gt
		}
		if cs.Gte == nil {
			scale["gte"] = math.MaxFloat32
		} else {
			scale["gte"] = *cs.Gte
		}
		if cs.Lt == nil {
			scale["lt"] = math.MaxFloat32
		} else {
			scale["lt"] = *cs.Lt
		}
		if cs.Lte == nil {
			scale["lte"] = math.MaxFloat32
		} else {
			scale["lte"] = *cs.Lte
		}
		if cs.PaletteIndex != nil {
			color, err := getNameFromChartColorsByIndex(int(*cs.PaletteIndex))
			if err != nil {
				return nil, err
			}
			scale["color"] = color
		}
		scales[i] = scale
	}
	return scales, nil
}

func singlevaluechartRead(d *schema.ResourceData, meta interface{}) error {
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

	return singlevaluechartAPIToTF(d, c)
}

func singlevaluechartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadSingleValueChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Single Value Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Single Value Chart Response: %v", c)

	d.SetId(c.Id)
	return singlevaluechartAPIToTF(d, c)
}

func singlevaluechartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
