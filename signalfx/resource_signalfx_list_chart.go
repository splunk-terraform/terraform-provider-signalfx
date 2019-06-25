package signalfx

import (
	"encoding/json"
	"fmt"
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
				Description: "(Metric by default) Must be \"Metric\" or \"Binary\"",
			},
			"color_by": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
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
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of properties that shouldn't be displayed in the chart legend (i.e. dimension names)",
			},
			"max_precision": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of digits to display when rounding values up or down",
			},
			"secondary_visualization": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
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
			"synced": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing.",
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
	log.Printf("[DEBUG] Create Payload: %s", string(debugOutput))

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

func listchartAPIToTF(d *schema.ResourceData, chart *chart.Chart) error {
	log.Printf("[DEBUG] Got Time Chart %v", chart)

	return nil
}

func listchartRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	path := fmt.Sprintf("%s/%s", CHART_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceRead(url, config.AuthToken, d)
}

func listchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadListChart(d)

	chart, err := config.Client.UpdateChart(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Update Response: %v", chart)

	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+d.Id())
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(chart.Id)
	return listchartAPIToTF(d, chart)
}

func listchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", CHART_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceDelete(url, config.AuthToken, d)
}
