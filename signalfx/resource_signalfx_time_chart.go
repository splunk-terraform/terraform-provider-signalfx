package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	chart "github.com/signalfx/signalfx-go/chart"
)

var PaletteColors = map[string]int{
	"gray":       0,
	"blue":       1,
	"azure":      2,
	"navy":       3,
	"brown":      4,
	"orange":     5,
	"yellow":     6,
	"magenta":    7,
	"purple":     8,
	"pink":       9,
	"violet":     10,
	"lilac":      11,
	"iris":       12,
	"emerald":    13,
	"green":      14,
	"aquamarine": 15,
}

var FullPaletteColors = map[string]int{
	"gray":        0,
	"blue":        1,
	"azure":       2,
	"navy":        3,
	"brown":       4,
	"orange":      5,
	"yellow":      6,
	"magenta":     7,
	"purple":      8,
	"pink":        9,
	"violet":      10,
	"lilac":       11,
	"iris":        12,
	"emerald":     13,
	"green":       14,
	"aquamarine":  15,
	"red":         16,
	"gold":        17,
	"greenyellow": 18,
	"chartreuse":  19,
	"jade":        20,
}

func resourceAxisMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		return migrateAxisStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAxisStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		return is, nil
	}
	if v, ok := is.Attributes["max_value"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f == math.MaxFloat64 {
			delete(is.Attributes, "max_value")
		}
	}
	if v, ok := is.Attributes["min_value"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f == -math.MaxFloat64 {
			delete(is.Attributes, "min_value")
		}
	}
	if v, ok := is.Attributes["low_watermark"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f == -math.MaxFloat64 {
			delete(is.Attributes, "low_watermark")
		}
	}
	if v, ok := is.Attributes["high_watermark"]; ok {
		if f, err := strconv.ParseFloat(v, 32); err == nil && f == math.MaxFloat64 {
			delete(is.Attributes, "high_watermark")
		}
	}
	return is, nil
}

func timeChartResource() *schema.Resource {
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
				Description: "Description of the chart",
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
				Description: "(Dimension by default) Must be \"Dimension\" or \"Metric\"",
				ValidateFunc: validation.StringInSlice([]string{
					"Metric", "Dimension", "Scale",
				}, false),
			},
			"minimum_resolution": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The minimum resolution (in seconds) to use for computing the underlying program",
			},
			"max_delay": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "How long (in seconds) to wait for late datapoints",
				ValidateFunc: validation.IntBetween(0, 900),
			},
			"timezone": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UTC",
				Description: "The property value is a string that denotes the geographic region associated with the time zone, (e.g. Australia/Sydney)",
			},
			"disable_sampling": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) If false, samples a subset of the output MTS, which improves UI performance",
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
			"axis_right": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					SchemaVersion: 1,
					MigrateState:  resourceAxisMigrateState,
					Schema: map[string]*schema.Schema{
						"min_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     -math.MaxFloat64,
							Description: "The minimum value for the right axis",
						},
						"max_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat64,
							Description: "The maximum value for the right axis",
						},
						"label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Label of the right axis",
						},
						"high_watermark": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat64,
							Description: "A line to draw as a high watermark",
						},
						"high_watermark_label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "A label to attach to the high watermark line",
						},
						"low_watermark": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     -math.MaxFloat64,
							Description: "A line to draw as a low watermark",
						},
						"low_watermark_label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "A label to attach to the low watermark line",
						},
						"watermarks": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"value": &schema.Schema{
										Type:        schema.TypeFloat,
										Required:    true,
										Description: "Axis value where the watermark line will be displayed",
									},
									"label": &schema.Schema{
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Label to display associated with the watermark line",
									},
								},
							},
						},
					},
				},
			},
			"axis_left": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					SchemaVersion: 1,
					MigrateState:  resourceAxisMigrateState,
					Schema: map[string]*schema.Schema{
						"min_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     -math.MaxFloat64,
							Description: "The minimum value for the left axis",
						},
						"max_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat64,
							Description: "The maximum value for the left axis",
						},
						"label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Label of the left axis",
						},
						"high_watermark": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat64,
							Description: "A line to draw as a high watermark",
						},
						"high_watermark_label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "A label to attach to the high watermark line",
						},
						"low_watermark": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     -math.MaxFloat64,
							Description: "A line to draw as a low watermark",
						},
						"low_watermark_label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "A label to attach to the low watermark line",
						},
						"watermarks": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"value": &schema.Schema{
										Type:        schema.TypeFloat,
										Required:    true,
										Description: "Axis value where the watermark line will be displayed",
									},
									"label": &schema.Schema{
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Label to display associated with the watermark line",
									},
								},
							},
						},
					},
				},
			},
			"axes_precision": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
				Description: "Force a specific number of significant digits in the y-axis",
			},
			"axes_include_zero": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Force y-axes to always show zero",
			},
			"on_chart_legend_dimension": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Dimension to show in the on-chart legend. On-chart legend is off unless a dimension is specified. Allowed: 'metric', 'plot_label' and any dimension.",
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
			"show_event_lines": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "(false by default) Whether vertical highlight lines should be drawn in the visualizations at times when events occurred",
			},
			"show_data_markers": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "(false by default) Show markers (circles) for each datapoint used to draw line or area charts",
			},
			"stacked": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) Whether area and bar charts in the visualization should be stacked",
			},
			"tags": &schema.Schema{
				Type:        schema.TypeList,
				Deprecated:  "signalfx_time_chart.tags is being removed in the next release",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the chart",
			},
			"plot_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "LineChart",
				Description: "(LineChart by default) The default plot display style for the visualization. Must be \"LineChart\", \"AreaChart\", \"ColumnChart\", or \"Histogram\"",
				ValidateFunc: validation.StringInSlice([]string{
					"AreaChart", "ColumnChart", "Histogram", "LineChart",
				}, false),
			},
			"histogram_options": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Options specific to Histogram charts",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"color_theme": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Base color theme to use for the graph.",
							ValidateFunc: validateFullPaletteColors,
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
						"axis": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "left",
							Description: "The Y-axis associated with values for this plot. Must be either \"right\" or \"left\". Defaults to \"left\".",
							ValidateFunc: validation.StringInSlice([]string{
								"left", "right",
							}, false),
						},
						"plot_type": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "(Chart plot_type by default) The visualization style to use. Must be \"LineChart\", \"AreaChart\", \"ColumnChart\", or \"Histogram\"",
							ValidateFunc: validation.StringInSlice([]string{
								"AreaChart", "ColumnChart", "Histogram", "LineChart",
							}, false),
						},
						"display_name": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.",
						},
						"value_unit": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes)",
							ValidateFunc: validateUnitTimeChart,
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
			"event_options": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Event display customization options, associated with a publish statement",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label used in the publish statement that displays the events you want to customize",
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
					},
				},
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    timeRangeV0().CoreConfigSchema().ImpliedType(),
				Upgrade: timeRangeStateUpgradeV0,
				Version: 0,
			},
		},

		Create: timechartCreate,
		Read:   timechartRead,
		Update: timechartUpdate,
		Delete: timechartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a time chart
*/
func getPayloadTimeChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	var tags []string
	if val, ok := d.GetOk("tags"); ok {
		tags := []string{}
		for _, tag := range val.([]interface{}) {
			tags = append(tags, tag.(string))
		}
	}

	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Tags:        tags,
	}

	viz := getTimeChartOptions(d)
	if axesOptions := getAxesOptions(d); len(axesOptions) > 0 {
		viz.Axes = axesOptions
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
	if eventOptions := getPerEventOptions(d); len(eventOptions) > 0 {
		viz.EventPublishLabelOptions = eventOptions
	}
	if onChartLegendDim, ok := d.GetOk("on_chart_legend_dimension"); ok {
		if onChartLegendDim == "metric" {
			onChartLegendDim = "sf_originatingMetric"
		} else if onChartLegendDim == "plot_label" {
			onChartLegendDim = "sf_metric"
		}
		viz.OnChartLegendOptions = &chart.LegendOptions{
			ShowLegend:        true,
			DimensionInLegend: onChartLegendDim.(string),
		}
	}
	payload.Options = viz

	return payload
}

func getPerSignalVizOptions(d *schema.ResourceData) []*chart.PublishLabelOptions {
	viz := d.Get("viz_options").(*schema.Set).List()
	vizList := make([]*chart.PublishLabelOptions, len(viz))
	for i, v := range viz {
		v := v.(map[string]interface{})
		item := &chart.PublishLabelOptions{
			Label: v["label"].(string),
		}
		if val, ok := v["display_name"].(string); ok && val != "" {
			item.DisplayName = val
		}
		if val, ok := v["color"].(string); ok {
			if elem, ok := PaletteColors[val]; ok {
				i := int32(elem)
				item.PaletteIndex = &i
			}
		}
		if val, ok := v["plot_type"].(string); ok && val != "" {
			item.PlotType = val
		}
		if val, ok := v["axis"].(string); ok && val != "" {
			if val == "right" {
				item.YAxis = int32(1)
			} else {
				item.YAxis = int32(0)
			}
		}
		if val, ok := v["value_unit"].(string); ok && val != "" {
			item.ValueUnit = val
		}
		if val, ok := v["value_suffix"].(string); ok && val != "" {
			item.ValueSuffix = val
		}
		if val, ok := v["value_prefix"].(string); ok && val != "" {
			item.ValuePrefix = val
		}

		vizList[i] = item
	}
	return vizList
}

func getPerEventOptions(d *schema.ResourceData) []*chart.EventPublishLabelOptions {
	eos := d.Get("event_options").(*schema.Set).List()
	eventList := make([]*chart.EventPublishLabelOptions, len(eos))
	for i, ev := range eos {
		ev := ev.(map[string]interface{})
		item := &chart.EventPublishLabelOptions{
			Label: ev["label"].(string),
		}
		if val, ok := ev["display_name"].(string); ok && val != "" {
			item.DisplayName = val
		}
		if val, ok := ev["color"].(string); ok {
			if elem, ok := PaletteColors[val]; ok {
				i := int32(elem)
				item.PaletteIndex = &i
			}
		}

		eventList[i] = item
	}
	return eventList
}

func getAxesOptions(d *schema.ResourceData) []*chart.Axes {
	axesListopts := make([]*chart.Axes, 2)
	if tfAxisOpts, ok := d.GetOk("axis_right"); ok {
		tfRightAxisOpts := tfAxisOpts.(*schema.Set).List()[0]
		tfOpt := tfRightAxisOpts.(map[string]interface{})
		axesListopts[1] = getSingleAxisOptions(tfOpt)
	}
	if tfAxisOpts, ok := d.GetOk("axis_left"); ok {
		tfLeftAxisOpts := tfAxisOpts.(*schema.Set).List()[0]
		tfOpt := tfLeftAxisOpts.(map[string]interface{})
		axesListopts[0] = getSingleAxisOptions(tfOpt)
	}
	return axesListopts
}

func getSingleAxisOptions(axisOpt map[string]interface{}) *chart.Axes {
	axis := &chart.Axes{}

	if val, ok := axisOpt["min_value"]; ok {
		axis.Min = getValueUsingMaxFloatAsDefault(val.(float64))
	}
	if val, ok := axisOpt["max_value"]; ok {
		axis.Max = getValueUsingMaxFloatAsDefault(val.(float64))
	}
	if val, ok := axisOpt["label"]; ok {
		axis.Label = val.(string)
	}
	if val, ok := axisOpt["high_watermark"]; ok {
		axis.HighWatermark = getValueUsingMaxFloatAsDefault(val.(float64))
	}
	if val, ok := axisOpt["high_watermark_label"]; ok {
		axis.HighWatermarkLabel = val.(string)
	}
	if val, ok := axisOpt["low_watermark"]; ok {
		axis.LowWatermark = getValueUsingMaxFloatAsDefault(val.(float64))
	}
	if val, ok := axisOpt["low_watermark_label"]; ok {
		axis.LowWatermarkLabel = val.(string)
	}
	if *axis == (chart.Axes{}) {
		// We set nothing, so return a nil
		return nil
	}
	return axis
}

func getTimeChartOptions(d *schema.ResourceData) *chart.Options {
	options := &chart.Options{
		Stacked: d.Get("stacked").(bool),
		Type:    "TimeSeriesChart",
	}
	if val, ok := d.GetOk("unit_prefix"); ok {
		options.UnitPrefix = val.(string)
	}
	if val, ok := d.GetOk("color_by"); ok {
		options.ColorBy = val.(string)
	}
	if val, ok := d.GetOk("show_event_lines"); ok {
		options.ShowEventLines = val.(bool)
	}
	if val, ok := d.GetOk("plot_type"); ok {
		options.DefaultPlotType = val.(string)
	}

	if val, ok := d.GetOk("axes_precision"); ok {
		ap := int32(val.(int))
		options.AxisPrecision = &ap
	}
	if val, ok := d.GetOk("axes_include_zero"); ok {
		options.IncludeZero = val.(bool)
	}

	var programOptions *chart.GeneralOptions
	if programOptions != (&chart.GeneralOptions{}) {
		programOptions = &chart.GeneralOptions{}
	}
	if val, ok := d.GetOk("minimum_resolution"); ok {
		mr := int32(val.(int) * 1000)
		programOptions.MinimumResolution = &mr
	}
	if val, ok := d.GetOk("max_delay"); ok {
		md := int32(val.(int) * 1000)
		programOptions.MaxDelay = &md
	}
	if val, ok := d.GetOk("timezone"); ok {
		programOptions.Timezone = val.(string)
	}
	if val, ok := d.GetOk("disable_sampling"); ok {
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

	// dataMarkersOption := make(map[string]interface{})
	showDataMarkers := d.Get("show_data_markers").(bool)
	if chartType, ok := d.GetOk("plot_type"); ok {
		chartType := chartType.(string)
		switch chartType {
		case "AreaChart":
			options.AreaChartOptions = &chart.AreaChartOptions{
				ShowDataMarkers: showDataMarkers,
			}
		case "Histogram":
			if histogramOptions, ok := d.GetOk("histogram_options"); ok {
				hOptions := histogramOptions.([]interface{})
				hOption := hOptions[0].(map[string]interface{})
				if colorTheme, ok := hOption["color_theme"].(string); ok {
					if elem, ok := FullPaletteColors[colorTheme]; ok {
						i := int32(elem)
						options.HistogramChartOptions = &chart.HistogramChartOptions{
							ColorThemeIndex: &i,
						}
					}
				}
			}
		// Not we don't have an option for LineChart as it is the same as
		// this default
		default:
			options.LineChartOptions = &chart.LineChartOptions{
				ShowDataMarkers: showDataMarkers,
			}
		}
	}

	return options
}

func timechartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadTimeChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Time Chart Payload: %s", string(debugOutput))

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

	return timechartAPIToTF(d, c)
}

func timechartRead(d *schema.ResourceData, meta interface{}) error {
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

	return timechartAPIToTF(d, c)
}

func timechartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Time Chart to enState: %s", string(debugOutput))

	if err := d.Set("name", c.Name); err != nil {
		return err
	}
	if err := d.Set("description", c.Description); err != nil {
		return err
	}
	if err := d.Set("program_text", c.ProgramText); err != nil {
		return err
	}
	if err := d.Set("tags", c.Tags); err != nil {
		return err
	}
	options := c.Options

	if err := d.Set("axes_include_zero", options.IncludeZero); err != nil {
		return err
	}
	if err := d.Set("color_by", options.ColorBy); err != nil {
		return err
	}
	if err := d.Set("plot_type", options.DefaultPlotType); err != nil {
		return err
	}
	if err := d.Set("axes_precision", options.AxisPrecision); err != nil {
		return err
	}
	if err := d.Set("show_event_lines", options.ShowEventLines); err != nil {
		return err
	}
	if err := d.Set("stacked", options.Stacked); err != nil {
		return err
	}
	if err := d.Set("unit_prefix", options.UnitPrefix); err != nil {
		return err
	}

	if options.AreaChartOptions != nil {
		if err := d.Set("show_data_markers", options.AreaChartOptions.ShowDataMarkers); err != nil {
			return err
		}
	}
	if options.LineChartOptions != nil {
		if err := d.Set("show_data_markers", options.LineChartOptions.ShowDataMarkers); err != nil {
			return err
		}
	}
	if options.HistogramChartOptions != nil {
		if options.HistogramChartOptions.ColorThemeIndex != nil {
			color, err := getNameFromFullPaletteColorsByIndex(int(*options.HistogramChartOptions.ColorThemeIndex))
			if err != nil {
				return err
			}
			histOptions := map[string]interface{}{
				"color_theme": color,
			}
			if err := d.Set("histogram_options", []interface{}{histOptions}); err != nil {
				return err
			}
		}
	}

	if len(options.Axes) > 0 {
		axisLeft := options.Axes[0]
		// We need to verify that there are real axes and not just nil
		// or zeroed structs, so we do comparison before setting each.
		if (axisLeft == nil || *axisLeft == chart.Axes{}) {
			log.Printf("[DEBUG] SignalFx: Axis Left is nil or zero, skipping")
		} else {
			if err := d.Set("axis_left", axisToMap(axisLeft)); err != nil {
				return err
			}
		}
		if len(options.Axes) > 1 {
			axisRight := options.Axes[1]
			if (axisRight == nil || *axisRight == chart.Axes{}) {
				log.Printf("[DEBUG] SignalFx: Axis Right is nil or zero, skipping")
			} else {
				log.Printf("[DEBUG] SignalFx: Axis Right is real: %v", axisRight)
				if err := d.Set("axis_right", axisToMap(axisRight)); err != nil {
					return err
				}
			}
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
		if err := d.Set("disable_sampling", options.ProgramOptions.DisableSampling); err != nil {
			return err
		}
		if err := d.Set("timezone", options.ProgramOptions.Timezone); err != nil {
			return err
		}
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
			no, err := publishLabelOptionsToMap(plo)
			if err != nil {
				return err
			}
			plos[i] = no
		}
		if err := d.Set("viz_options", plos); err != nil {
			return err
		}
	}

	if len(options.EventPublishLabelOptions) > 0 {
		eplos := make([]map[string]interface{}, len(options.EventPublishLabelOptions))
		for i, eplo := range options.EventPublishLabelOptions {
			color := ""
			if eplo.PaletteIndex != nil {
				// We might not have a color, so tread lightly
				c, err := getNameFromPaletteColorsByIndex(int(*eplo.PaletteIndex))
				if err != nil {
					return err
				}
				// Ok, we can set the color now
				color = c
			}
			eplos[i] = map[string]interface{}{
				"label":        eplo.Label,
				"display_name": eplo.DisplayName,
				"color":        color,
			}
		}
		if err := d.Set("event_options", eplos); err != nil {
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

	if options.OnChartLegendOptions != nil {
		dil := options.OnChartLegendOptions.DimensionInLegend
		onChartLegendDim := dil
		// We use different names inside TF, so convert them back
		currDim := d.Get("on_chart_legend_dimension").(string)
		if dil == "sf_originatingMetric" && currDim != "sf_originatingMetric" {
			onChartLegendDim = "metric"
		} else if dil == "sf_metric" && currDim != "sf_metric" {
			onChartLegendDim = "plot_label"
		}
		if err := d.Set("on_chart_legend_dimension", onChartLegendDim); err != nil {
			return err
		}
	}

	return nil
}

func axisToMap(axis *chart.Axes) []*map[string]interface{} {
	if axis != nil {
		// We have to deal with a few defaults
		hwm := math.MaxFloat64
		if axis.HighWatermark != nil {
			hwm = float64(*axis.HighWatermark)
		}
		lwm := -math.MaxFloat64
		if axis.LowWatermark != nil {
			lwm = float64(*axis.LowWatermark)
		}
		max := math.MaxFloat64
		if axis.Max != nil {
			max = float64(*axis.Max)
		}
		min := -math.MaxFloat64
		if axis.Min != nil {
			min = float64(*axis.Min)
		}

		return []*map[string]interface{}{
			&map[string]interface{}{
				"high_watermark":       hwm,
				"high_watermark_label": axis.HighWatermarkLabel,
				"label":                axis.Label,
				"low_watermark":        lwm,
				"low_watermark_label":  axis.LowWatermarkLabel,
				"max_value":            max,
				"min_value":            min,
			},
		}
	}
	return nil
}

// This function handles a LabelOptions for non-time charts.
func publishNonTimeLabelOptionsToMap(options *chart.PublishLabelOptions) (map[string]interface{}, error) {
	color := ""
	if options.PaletteIndex != nil {
		// We might not have a color, so tread lightly
		c, err := getNameFromPaletteColorsByIndex(int(*options.PaletteIndex))
		if err != nil {
			return map[string]interface{}{}, err
		}
		// Ok, we can set the color now
		color = c
	}

	return map[string]interface{}{
		"label":        options.Label,
		"display_name": options.DisplayName,
		"color":        color,
		"value_unit":   options.ValueUnit,
		"value_suffix": options.ValueSuffix,
		"value_prefix": options.ValuePrefix,
	}, nil
}

func publishLabelOptionsToMap(options *chart.PublishLabelOptions) (map[string]interface{}, error) {
	color := ""
	if options.PaletteIndex != nil {
		// We might not have a color, so tread lightly
		c, err := getNameFromPaletteColorsByIndex(int(*options.PaletteIndex))
		if err != nil {
			return map[string]interface{}{}, err
		}
		// Ok, we can set the color now
		color = c
	}
	axis := "left"
	if options.YAxis == 1 {
		axis = "right"
	}

	return map[string]interface{}{
		"label":        options.Label,
		"display_name": options.DisplayName,
		"color":        color,
		"axis":         axis,
		"plot_type":    options.PlotType,
		"value_unit":   options.ValueUnit,
		"value_suffix": options.ValueSuffix,
		"value_prefix": options.ValuePrefix,
	}, nil
}

func timechartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadTimeChart(d)

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Time Chart Response: %v", c)

	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return timechartAPIToTF(d, c)
}

func timechartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}

var validateUnitTimeChart = validation.StringInSlice([]string{
	"Bit",
	"Kilobit",
	"Megabit",
	"Gigabit",
	"Terabit",
	"Petabit",
	"Exabit",
	"Zettabit",
	"Yottabit",
	"Byte",
	"Kibibyte",
	"Mebibyte",
	"Gigibyte",
	"Tebibyte",
	"Pebibyte",
	"Exbibyte",
	"Zebibyte",
	"Yobibyte",
	"Nanosecond",
	"Microsecond",
	"Millisecond",
	"Second",
	"Minute",
	"Hour",
	"Day",
	"Week",
}, false)
