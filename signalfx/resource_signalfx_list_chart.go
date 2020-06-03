package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	chart "github.com/signalfx/signalfx-go/chart"
)

func listChartResource() *schema.Resource {
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
				Default:     "Dimension",
				Description: "(Metric by default) Must be \"Scale\", \"Metric\" or \"Dimension\"",
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
			"disable_sampling": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) If false, samples a subset of the output MTS, which improves UI performance",
			},
			"sort_by": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSortBy,
				Description:  "The property to use when sorting the elements. Use 'value' if you want to sort by value. Must be prepended with + for ascending or - for descending (e.g. -foo)",
			},
			"refresh_interval": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "How often (in seconds) to refresh the values of the list",
			},
			"legend_fields_to_hide": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				Deprecated:    "Please use legend_options_fields",
				ConflictsWith: []string{"legend_options_fields"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Description:   "List of properties that shouldn't be displayed in the chart legend (i.e. dimension names)",
			},
			"legend_options_fields": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of a property to hide or show in the data table.",
						},
						"enabled": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "(true by default) Determines if this property is displayed in the data table.",
						},
					},
				},
				Optional:      true,
				ConflictsWith: []string{"legend_fields_to_hide"},
				Description:   "List of property and enabled flags to control the order and presence of datatable labels in a chart.",
			},
			"max_precision": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of digits to display when rounding values up or down",
			},
			"secondary_visualization": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Sparkline",
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
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: listchartCreate,
		Read:   listchartRead,
		Update: listchartUpdate,
		Delete: listchartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a list chart
*/
func getPayloadListChart(d *schema.ResourceData) (*chart.CreateUpdateChartRequest, error) {
	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
	}

	viz, err := getListChartOptions(d)
	if err != nil {
		return nil, err
	}
	// There are two ways to maniplate the legend. The first is keyed from
	// `legend_fields_to_hide`. Anything in this is marked as hidden. Unspecified
	// fields default to showing up in SFx's UI.
	if legendOptions := getLegendOptions(d); legendOptions != nil {
		viz.LegendOptions = legendOptions
		// Alternatively, the `legend_options_fields` provides finer control,
		// allowing ordering and on/off toggles. This is preferred, but we keep
		// `legend_fields_to_hide` for convenience.
	} else if legendOptions := getLegendFieldOptions(d); legendOptions != nil {
		viz.LegendOptions = legendOptions
	}

	if vizOptions := getPerSignalVizOptions(d); len(vizOptions) > 0 {
		viz.PublishLabelOptions = vizOptions
	}
	payload.Options = viz

	return payload, nil
}

func getListChartOptions(d *schema.ResourceData) (*chart.Options, error) {
	options := &chart.Options{
		Type: "List",
	}
	if val, ok := d.GetOk("unit_prefix"); ok {
		options.UnitPrefix = val.(string)
	}
	if val, ok := d.GetOk("color_by"); ok {
		options.ColorBy = val.(string)
		if val == "Scale" {
			if colorScaleOptions := getColorScaleOptions(d); len(colorScaleOptions) > 0 {
				options.ColorScale2 = colorScaleOptions
			}
		} else {
			if val, ok := d.GetOk("color_scale"); ok && val != nil {
				return nil, fmt.Errorf("Using `color_scale` without `color_by = \"Scale\"` has no effect")
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
	if val, ok := d.GetOk("disable_sampling"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.DisableSampling = val.(bool)
	}
	options.ProgramOptions = programOptions

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
	options.Time = timeOptions

	if sortBy, ok := d.GetOk("sort_by"); ok {
		options.SortBy = sortBy.(string)
	}
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

	return options, nil
}

func listchartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadListChart(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create List Chart Payload: %s", string(debugOutput))

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
	return listchartAPIToTF(d, c)
}

func listchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got List Chart to enState: %s", string(debugOutput))

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
	if options.ColorBy == "Scale" && len(options.ColorScale2) > 0 {
		colorScale, err := decodeColorScale(options)
		if err != nil {
			return err
		}
		if err := d.Set("color_scale", colorScale); err != nil {
			return err
		}
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
	if err := d.Set("sort_by", options.SortBy); err != nil {
		return err
	}

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

	if options.LegendOptions != nil && len(options.LegendOptions.Fields) > 0 {
		fields := make([]map[string]interface{}, len(options.LegendOptions.Fields))
		for i, lo := range options.LegendOptions.Fields {
			fields[i] = map[string]interface{}{
				"property": lo.Property,
				"enabled":  lo.Enabled,
			}
		}
		if err := d.Set("legend_options_fields", fields); err != nil {
			return err
		}
	}

	if options.ProgramOptions != nil {
		if options.ProgramOptions.MaxDelay != nil {
			if err := d.Set("max_delay", *options.ProgramOptions.MaxDelay/1000); err != nil {
				return err
			}
		}
		if err := d.Set("disable_sampling", options.ProgramOptions.DisableSampling); err != nil {
			return err
		}
	}

	return nil
}

func listchartRead(d *schema.ResourceData, meta interface{}) error {
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

	return listchartAPIToTF(d, c)
}

func listchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadListChart(d)
	if err != nil {
		return err
	}
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update List Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update List Chart Response: %v", c)

	d.SetId(c.Id)
	return listchartAPIToTF(d, c)
}

func listchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}
