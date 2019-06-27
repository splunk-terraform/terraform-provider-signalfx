package signalfx

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
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
				Type:        schema.TypeString,
				Required:    true,
				Description: "Signalflow program text for the chart. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
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
			},
			"color_by": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Dimension",
				Description: "(Metric by default) Must be \"Metric\" or \"Dimension\"",
			},
			"max_delay": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "How long (in seconds) to wait for late datapoints",
				ValidateFunc: validateMaxDelayValue,
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
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
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
	}
}

/*
  Use Resource object to construct json payload in order to create a list chart
*/
func getPayloadListChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
	}

	viz := getListChartOptions(d)
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

	return payload
}

func getListChartOptions(d *schema.ResourceData) *chart.Options {
	options := &chart.Options{
		Type: "List",
	}
	if val, ok := d.GetOk("unit_prefix"); ok {
		options.UnitPrefix = val.(string)
	}
	if val, ok := d.GetOk("color_by"); ok {
		options.ColorBy = val.(string)
	}

	var programOptions *chart.GeneralOptions
	if val, ok := d.GetOk("max_delay"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.MaxDelay = int32(val.(int) * 1000)
	}
	if val, ok := d.GetOk("disable_sampling"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.DisableSampling = val.(bool)
	}
	options.ProgramOptions = programOptions

	if sortBy, ok := d.GetOk("sort_by"); ok {
		options.SortBy = sortBy.(string)
	}
	if refreshInterval, ok := d.GetOk("refresh_interval"); ok {
		options.RefreshInterval = int32(refreshInterval.(int) * 1000)
	}
	if maxPrecision, ok := d.GetOk("max_precision"); ok {
		options.MaximumPrecision = int32(maxPrecision.(int))
	}
	if val, ok := d.GetOk("secondary_visualization"); ok {
		secondaryVisualization := val.(string)
		if secondaryVisualization != "" {
			options.SecondaryVisualization = secondaryVisualization
		}
	}

	return options
}

func listchartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadListChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create List Chart Payload: %s", string(debugOutput))

	chart, err := config.Client.CreateChart(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+d.Id())
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(chart.Id)
	return listchartAPIToTF(d, chart)
}

func listchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	log.Printf("[DEBUG] SignalFx: Got List Chart to enState %v", c)

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
	if err := d.Set("refresh_interval", options.RefreshInterval/1000); err != nil {
		return err
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
		if err := d.Set("max_delay", options.ProgramOptions.MaxDelay/1000); err != nil {
			return err
		}
		if err := d.Set("disable_sampling", options.ProgramOptions.DisableSampling); err != nil {
			return err
		}
	}

	return nil
}

func listchartRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	chart, err := config.Client.GetChart(d.Id())
	if err != nil {
		return err
	}

	return listchartAPIToTF(d, chart)
}

func listchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadListChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update List Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update List Chart Response: %v", c)

	d.SetId(c.Id)
	return listchartAPIToTF(d, c)
}

func listchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(d.Id())
}
