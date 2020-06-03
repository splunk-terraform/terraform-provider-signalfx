package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	chart "github.com/signalfx/signalfx-go/chart"
)

var validHexColor = regexp.MustCompile("^#[A-Fa-f0-9]{6}")

func heatmapChartResource() *schema.Resource {
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
			"refresh_interval": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "How often (in seconds) to refresh the values of the heatmap",
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
				Description: "Properties to group by in the heatmap (in nesting order)",
			},
			"sort_by": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSortBy,
				Description:  "The property to use when sorting the elements. Must be prepended with + for ascending or - for descending (e.g. -foo)",
			},
			"color_range": &schema.Schema{
				Type:          schema.TypeSet,
				MaxItems:      1,
				Optional:      true,
				ConflictsWith: []string{"color_scale"},
				Description:   "Values and color for the color range. Example: colorRange : { min : 0, max : 100, color : \"#0000ff\" }",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"color": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringMatch(validHexColor, "does not look like a hex color, similar to #0000ff"),
							Description:  "The color range to use. The starting hex color value for data values in a heatmap chart. Specify the value as a 6-character hexadecimal value preceded by the '#' character, for example \"#ea1849\" (grass green).",
						},
						"min_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     -math.MaxFloat32,
							Description: "The minimum value within the coloring range",
						},
						"max_value": &schema.Schema{
							Type:        schema.TypeFloat,
							Optional:    true,
							Default:     math.MaxFloat32,
							Description: "The maximum value within the coloring range",
						},
					},
				},
			},
			"color_scale": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"color_range"},
				Description:   "Single color range including both the color to display for that range and the borders of the range",
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
		},

		Create: heatmapchartCreate,
		Read:   heatmapchartRead,
		Update: heatmapchartUpdate,
		Delete: heatmapchartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create an Heatmap chart
*/
func getPayloadHeatmapChart(d *schema.ResourceData) (*chart.CreateUpdateChartRequest, error) {
	payload := &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
	}

	options, err := getHeatmapOptionsChart(d)
	if err != nil {
		return nil, err
	}
	payload.Options = options

	return payload, nil
}

func getHeatmapColorRangeOptions(d *schema.ResourceData) *chart.HeatmapColorRangeOptions {
	colorRange := d.Get("color_range").(*schema.Set).List()

	var item *chart.HeatmapColorRangeOptions

	for _, options := range colorRange {
		options := options.(map[string]interface{})

		// Don't make an empty color range.
		if options["color"].(string) == "" {
			return item
		}
		item = &chart.HeatmapColorRangeOptions{}

		if val, ok := options["min_value"]; ok {
			if val.(float64) != -math.MaxFloat32 {
				item.Min = val.(float64)
			}
		}
		if val, ok := options["max_value"]; ok {
			if val.(float64) != math.MaxFloat32 {
				item.Max = val.(float64)
			}
		}
		item.Color = options["color"].(string)
	}
	return item
}

func getHeatmapOptionsChart(d *schema.ResourceData) (*chart.Options, error) {
	options := &chart.Options{
		Type: "Heatmap",
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
	if val, ok := d.GetOk("disable_sampling"); ok {
		if programOptions == nil {
			programOptions = &chart.GeneralOptions{}
		}
		programOptions.DisableSampling = val.(bool)
	}
	options.ProgramOptions = programOptions

	if sortProperty, ok := d.GetOk("sort_by"); ok {
		sortBy := sortProperty.(string)
		options.SortProperty = sortBy[1:]
		if strings.HasPrefix(sortBy, "+") {
			options.SortDirection = "Ascending"
		} else {
			options.SortDirection = "Descending"
		}
	}

	// Default to an empty range
	options.ColorBy = "Range"
	if colorRangeOptions := getHeatmapColorRangeOptions(d); colorRangeOptions != nil {
		options.ColorBy = "Range"
		options.ColorRange = colorRangeOptions
	} else if colorScaleOptions := getColorScaleOptions(d); colorScaleOptions != nil && len(colorScaleOptions) > 0 {
		options.ColorBy = "Scale"
		options.ColorScale2 = colorScaleOptions
	}

	return options, nil
}

func heatmapchartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadHeatmapChart(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Heatmap Chart Payload: %s", string(debugOutput))

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

	return heatmapchartAPIToTF(d, c)
}

func heatmapchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Heatmap Chart to enState: %s", string(debugOutput))

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
		if err := d.Set("disable_sampling", options.ProgramOptions.DisableSampling); err != nil {
			return err
		}
	}

	if options.SortProperty != "" {
		sortBy := fmt.Sprintf("+%s", options.SortProperty)
		if options.SortDirection == "Descending" {
			sortBy = fmt.Sprintf("-%s", options.SortProperty)
		}
		if err := d.Set("sort_by", sortBy); err != nil {
			return err
		}
	}

	return nil
}

func heatmapchartRead(d *schema.ResourceData, meta interface{}) error {
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

	return heatmapchartAPIToTF(d, c)
}

func heatmapchartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadHeatmapChart(d)
	if err != nil {
		return err
	}

	c, err := config.Client.UpdateChart(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Heatmap Chart Response: %v", c)

	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return heatmapchartAPIToTF(d, c)
}

func heatmapchartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(context.TODO(), d.Id())
}

/*
  Validates the color_range field against a list of allowed words.
*/
func validateHeatmapChartColor(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	keys := make([]string, 0, len(ChartColorsSlice))
	found := false
	for _, item := range ChartColorsSlice {
		if value == item.name {
			found = true
		}
		keys = append(keys, item.name)
	}
	if !found {
		joinedColors := strings.Join(keys, ",")
		errors = append(errors, fmt.Errorf("%s not allowed; must be either %s", value, joinedColors))
	}
	return
}
