package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/signalfx/signalfx-go/dashboard"
	"github.com/signalfx/signalfx-go/util"
)

const (
	DashboardAppPath = "/dashboard/"
)

func dashboardResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the dashboard",
			},
			"dashboard_group": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the dashboard group that contains the dashboard. If an ID is not provided during creation, the dashboard will be placed in a newly created dashboard group",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the dashboard (Optional)",
			},
			"charts_resolution": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      strings.ToLower(string(dashboard.DEFAULT)),
				Description:  "Specifies the chart data display resolution for charts in this dashboard. Value can be one of \"default\", \"low\", \"high\", or \"highest\". default by default",
				ValidateFunc: validateChartsResolution,
			},
			"time_range": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validateSignalfxRelativeTime,
				Description:   "From when to display data. SignalFx time syntax (e.g. -5m, -1h)",
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
			"chart": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"column", "grid"},
				Description:   "Chart ID and layout information for the charts in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"chart_id": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the chart to display",
						},
						"row": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
							Description:  "The row to show the chart in (zero-based); if height > 1, this value represents the topmost row of the chart. (greater than or equal to 0)",
						},
						"column": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 11),
							Description:  "The column to show the chart in (zero-based); this value always represents the leftmost column of the chart. (between 0 and 11)",
						},
						"width": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      12,
							ValidateFunc: validation.IntBetween(1, 12),
							Description:  "How many columns (out of a total of 12, one-based) the chart should take up. (between 1 and 12)",
						},
						"height": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
							Description:  "How many rows the chart should take up. (greater than or equal to 1)",
						},
					},
				},
			},
			"grid": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"column", "chart"},
				Description:   "Grid dashboard layout. Charts listed will be placed in a grid by row with the same width and height. If a chart can't fit in a row, it will be placed automatically in the next row",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"chart_ids": &schema.Schema{
							Type:        schema.TypeList,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Charts to use for the grid",
						},
						"width": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      12,
							ValidateFunc: validation.IntBetween(1, 12),
							Description:  "Number of columns (out of a total of 12, one-based) each chart should take up. (between 1 and 12)",
						},
						"height": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
							Description:  "How many rows each chart should take up. (greater than or equal to 1)",
						},
					},
				},
			},
			"column": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"grid", "chart"},
				Description:   "Column layout. Charts listed, will be placed in a single column with the same width and height",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"chart_ids": &schema.Schema{
							Type:        schema.TypeList,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Charts to use for the column",
						},
						"column": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 11),
							Description:  "The column to show the chart in (zero-based); this value always represents the leftmost column of the chart. (between 0 and 11)",
						},
						"width": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      12,
							ValidateFunc: validation.IntBetween(1, 12),
							Description:  "Number of columns (out of a total of 12) each chart should take up. (between 1 and 12)",
						},
						"height": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
							Description:  "How many rows each chart should take up. (greater than or equal to 1)",
						},
					},
				},
			},
			"variable": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Dashboard variable to apply to each chart in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "A metric time series dimension or property name",
						},
						"alias": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "An alias for the dashboard variable. This text will appear as the label for the dropdown field on the dashboard",
						},
						"description": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Variable description",
						},
						"values": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of strings (which will be treated as an OR filter on the property)",
						},
						"value_required": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Determines whether a value is required for this variable (and therefore whether it will be possible to view this dashboard without this filter applied). false by default",
						},
						"values_suggested": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "A list of strings of suggested values for this variable; these suggestions will receive priority when values are autosuggested for this variable",
						},
						"restricted_suggestions": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, this variable may only be set to the values listed in preferredSuggestions. and only these values will appear in autosuggestion menus. false by default",
						},
						"replace_only": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, this variable will only apply to charts with a filter on the named property.",
						},
						"apply_if_exist": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, this variable will also match data that does not have the specified property",
						},
					},
				},
			},
			"filter": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Filter to apply to each chart in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "A metric time series dimension or property name",
						},
						"values": &schema.Schema{
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of strings (which will be treated as an OR filter on the property)",
						},
						"negated": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "(false by default) Whether this filter should be a \"not\" filter",
						},
						"apply_if_exist": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, this filter will also match data that does not have the specified property",
						},
					},
				},
			},
			"event_overlay": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Event overlay to add to charts",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"signal": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Search term used to define events",
						},
						"line": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "(false by default) Whether a vertical line should be displayed in the plot at the time the event occurs",
						},
						"label": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The text displaying in the dropdown menu used to select this event overlay as an active overlay for the dashboard.",
						},
						"color": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Color to use",
							ValidateFunc: validatePerSignalColor,
						},
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "eventTimeSeries",
							Description:  "Source for this event's data. Can be \"eventTimeSeries\" (default) or \"detectorEvents\".",
							ValidateFunc: validateEventOverlayType,
						},
						"source": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "A metric time series dimension or property name",
									},
									"values": &schema.Schema{
										Type:        schema.TypeSet,
										Required:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of strings (which will be treated as an OR filter on the property)",
									},
									"negated": &schema.Schema{
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "(false by default) Whether this filter should be a \"not\" filter",
									},
								},
							},
						},
					},
				},
			},
			"selected_event_overlay": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Event overlay added to charts by default to charts",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"signal": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Search term used to define events",
						},
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "eventTimeSeries",
							Description:  "Source for this event's data. Can be \"eventTimeSeries\" (default) or \"detectorEvents\".",
							ValidateFunc: validateEventOverlayType,
						},
						"source": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "A metric time series dimension or property name",
									},
									"values": &schema.Schema{
										Type:        schema.TypeSet,
										Required:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of strings (which will be treated as an OR filter on the property)",
									},
									"negated": &schema.Schema{
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "(false by default) Whether this filter should be a \"not\" filter",
									},
								},
							},
						},
					},
				},
			},
			"authorized_writer_teams": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs that have write access to this dashboard",
			},
			"authorized_writer_users": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "User IDs that have write access to this dashboard",
			},
			"discovery_options_query": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"discovery_options_selectors": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the dashboard",
			},
		},

		Create: dashboardCreate,
		Read:   dashboardRead,
		Update: dashboardUpdate,
		Delete: dashboardDelete,
		Exists: dashboardExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create a dashboard
*/
func getPayloadDashboard(d *schema.ResourceData) (*dashboard.CreateUpdateDashboardRequest, error) {

	cudr := &dashboard.CreateUpdateDashboardRequest{
		Name:              d.Get("name").(string),
		Description:       d.Get("description").(string),
		GroupId:           d.Get("dashboard_group").(string),
		AuthorizedWriters: &dashboard.AuthorizedWriters{},
	}

	if val, ok := d.GetOk("authorized_writer_teams"); ok {
		var teams []string
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			teams = append(teams, v.(string))
		}
		cudr.AuthorizedWriters.Teams = teams
	}
	if val, ok := d.GetOk("authorized_writer_users"); ok {
		var users []string
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			users = append(users, v.(string))
		}
		cudr.AuthorizedWriters.Users = users
	}

	allFilters := &dashboard.ChartsFilters{}
	if filters := getDashboardFilters(d); len(filters) > 0 {
		allFilters.Sources = filters
	}
	if variables := getDashboardVariables(d); len(variables) > 0 {
		allFilters.Variables = variables
	}
	allFilters.Time = getDashboardTime(d)
	cudr.Filters = allFilters

	overlays := d.Get("event_overlay").([]interface{})
	cudr.EventOverlays = getDashboardEventOverlays(overlays)

	if soverlays, ok := d.GetOk("selected_event_overlay"); ok {
		soverlays := soverlays.([]interface{})
		cudr.SelectedEventOverlays = getDashboardEventOverlays(soverlays)
	}

	charts := getDashboardCharts(d)
	columnCharts := getDashboardColumns(d)
	dashboardCharts := append(charts, columnCharts...)
	gridCharts := getDashboardGrids(d)
	dashboardCharts = append(dashboardCharts, gridCharts...)
	if len(dashboardCharts) > 0 {
		cudr.Charts = dashboardCharts
	}

	if chartsResolution, ok := d.GetOk("charts_resolution"); ok {
		density := strings.ToUpper(chartsResolution.(string))
		switch density {
		case "LOW":
			cudr.ChartDensity = dashboard.LOW
		case "HIGH":
			cudr.ChartDensity = dashboard.HIGH
		case "HIGHEST":
			cudr.ChartDensity = dashboard.HIGHEST
		default:
			cudr.ChartDensity = dashboard.DEFAULT
		}
	}

	if doQuery, ok := d.GetOk("discovery_options_query"); ok {
		var selectors []string
		if val, ok := d.GetOk("discovery_options_selectors"); ok {
			tfSels := val.(*schema.Set).List()
			for _, v := range tfSels {
				selectors = append(selectors, v.(string))
			}
		}
		cudr.DiscoveryOptions = &dashboard.DiscoveryOptions{
			Query:     doQuery.(string),
			Selectors: &selectors,
		}
	}

	return cudr, nil
}

func getDashboardTime(d *schema.ResourceData) *dashboard.ChartsFiltersTime {
	var timeFilter *dashboard.ChartsFiltersTime
	if val, ok := d.GetOk("time_range"); ok {
		timeFilter = &dashboard.ChartsFiltersTime{
			Start: util.StringOrInteger(val.(string)),
			End:   "Now",
		}
	} else {
		if val, ok := d.GetOk("start_time"); ok {
			timeFilter = &dashboard.ChartsFiltersTime{
				Start: util.StringOrInteger(strconv.Itoa(val.(int) * 1000)),
			}
			if val, ok := d.GetOk("end_time"); ok {
				timeFilter.End = util.StringOrInteger(strconv.Itoa(val.(int) * 1000))
			}
		}
	}
	return timeFilter
}

func getDashboardCharts(d *schema.ResourceData) []*dashboard.DashboardChart {
	charts := d.Get("chart").(*schema.Set).List()
	chartsList := make([]*dashboard.DashboardChart, len(charts))
	for i, chart := range charts {
		chart := chart.(map[string]interface{})
		item := &dashboard.DashboardChart{
			ChartId: chart["chart_id"].(string),
			Column:  int32(chart["column"].(int)),
			Height:  int32(chart["height"].(int)),
			Row:     int32(chart["row"].(int)),
			Width:   int32(chart["width"].(int)),
		}

		chartsList[i] = item
	}
	return chartsList
}

func getDashboardColumns(d *schema.ResourceData) []*dashboard.DashboardChart {
	columns := d.Get("column").([]interface{})
	charts := make([]*dashboard.DashboardChart, 0)
	for _, column := range columns {
		column := column.(map[string]interface{})

		currentRow := 0
		columnNumber := column["column"].(int)
		for _, chartID := range column["chart_ids"].([]interface{}) {
			item := &dashboard.DashboardChart{
				ChartId: chartID.(string),
				Column:  int32(columnNumber),
				Height:  int32(column["height"].(int)),
				Row:     int32(currentRow),
				Width:   int32(column["width"].(int)),
			}

			currentRow++
			charts = append(charts, item)
		}
	}
	return charts
}

func getDashboardGrids(d *schema.ResourceData) []*dashboard.DashboardChart {
	grids := d.Get("grid").([]interface{})
	charts := make([]*dashboard.DashboardChart, 0)
	// We must keep track of the row outside of the loop as there might be many
	// grids to draw.
	currentRow := 0
	for _, grid := range grids {
		grid := grid.(map[string]interface{})

		width := grid["width"].(int)
		currentColumn := 0
		for _, chartID := range grid["chart_ids"].([]interface{}) {
			if currentColumn+width > 12 {
				currentRow++
				currentColumn = 0
			}

			item := &dashboard.DashboardChart{
				ChartId: chartID.(string),
				Column:  int32(currentColumn),
				Height:  int32(grid["height"].(int)),
				Row:     int32(currentRow),
				Width:   int32(grid["width"].(int)),
			}
			currentColumn += width
			charts = append(charts, item)
		}
		currentRow++ // Increment the row for the next grid
	}
	return charts
}

func getDashboardVariables(d *schema.ResourceData) []*dashboard.ChartsWebUiFilter {
	variables := d.Get("variable").(*schema.Set).List()
	varsList := make([]*dashboard.ChartsWebUiFilter, len(variables))
	for i, variable := range variables {
		variable := variable.(map[string]interface{})

		var values []string
		if val, ok := variable["values"]; ok {
			tfValues := val.(*schema.Set).List()
			for _, v := range tfValues {
				values = append(values, v.(string))
			}
		}

		var preferredSuggestions []string
		if val, ok := variable["values_suggested"]; ok {
			tfValues := val.(*schema.Set).List()
			for _, v := range tfValues {
				preferredSuggestions = append(preferredSuggestions, v.(string))
			}
		}

		item := &dashboard.ChartsWebUiFilter{
			Alias:                variable["alias"].(string),
			ApplyIfExists:        variable["apply_if_exist"].(bool),
			Description:          variable["description"].(string),
			PreferredSuggestions: preferredSuggestions,
			Property:             variable["property"].(string),
			Required:             variable["value_required"].(bool),
			ReplaceOnly:          variable["replace_only"].(bool),
			Restricted:           variable["restricted_suggestions"].(bool),
			Value:                values,
		}

		varsList[i] = item
	}
	return varsList
}

func getDashboardEventOverlays(overlays []interface{}) []*dashboard.ChartEventOverlay {
	overlayList := make([]*dashboard.ChartEventOverlay, len(overlays))
	for i, overlay := range overlays {
		overlay := overlay.(map[string]interface{})
		item := &dashboard.ChartEventOverlay{
			EventSignal: &dashboard.DashboardEventSignal{
				EventSearchText: overlay["signal"].(string),
				EventType:       overlay["type"].(string),
			},
		}
		if val, ok := overlay["line"].(bool); ok {
			item.EventLine = val
		}
		if val, ok := overlay["label"].(string); ok {
			item.Label = val
		}

		if val, ok := overlay["color"].(string); ok {
			if elem, ok := PaletteColors[val]; ok {
				i := int32(elem)
				item.EventColorIndex = &i
			}
		}

		if sources, ok := overlay["source"].([]interface{}); ok {
			item.Sources = getDashboardEventOverlayFilters(sources)
		}

		overlayList[i] = item
	}
	return overlayList
}

func getDashboardEventOverlayFilters(sources []interface{}) []*dashboard.EventOverlayFilter {
	sourcesList := make([]*dashboard.EventOverlayFilter, len(sources))
	for j, source := range sources {
		source := source.(map[string]interface{})

		tfValues := source["values"].(*schema.Set).List()
		values := make([]string, len(tfValues))
		for i, v := range tfValues {
			values[i] = v.(string)
		}

		s := &dashboard.EventOverlayFilter{
			NOT:      source["negated"].(bool),
			Property: source["property"].(string),
			Value:    values,
		}
		sourcesList[j] = s
	}
	return sourcesList
}

func getDashboardFilters(d *schema.ResourceData) []*dashboard.ChartsSingleFilter {
	filters := d.Get("filter").(*schema.Set).List()
	filterList := make([]*dashboard.ChartsSingleFilter, len(filters))
	for i, filter := range filters {
		filter := filter.(map[string]interface{})

		var values []string
		tfValues := filter["values"].(*schema.Set).List()
		if len(tfValues) > 0 {
			values = []string{}
			for _, v := range tfValues {
				values = append(values, v.(string))
			}
		}

		item := &dashboard.ChartsSingleFilter{
			NOT:           filter["negated"].(bool),
			Property:      filter["property"].(string),
			Value:         values,
			ApplyIfExists: filter["apply_if_exist"].(bool),
		}

		filterList[i] = item
	}
	return filterList
}

func dashboardCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Dashboard Create Payload: %s", debugOutput)

	dash, err := config.Client.CreateDashboard(context.TODO(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DashboardAppPath+dash.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(dash.Id)

	return dashboardAPIToTF(d, dash)
}

func dashboardExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetDashboard(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func dashboardRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	dash, err := config.Client.GetDashboard(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	appURL, err := buildAppURL(config.CustomAppURL, DashboardAppPath+dash.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}

	return dashboardAPIToTF(d, dash)
}

func dashboardAPIToTF(d *schema.ResourceData, dash *dashboard.Dashboard) error {
	debugOutput, _ := json.Marshal(dash)
	log.Printf("[DEBUG] SignalFx: Got Dashboard to enState: %s", string(debugOutput))

	if err := d.Set("name", dash.Name); err != nil {
		return err
	}
	if err := d.Set("dashboard_group", dash.GroupId); err != nil {
		return err
	}
	if err := d.Set("description", dash.Description); err != nil {
		return err
	}
	if err := d.Set("charts_resolution", strings.ToLower(string(*dash.ChartDensity))); err != nil {
		return err
	}

	if dash.AuthorizedWriters != nil {
		aw := dash.AuthorizedWriters
		if aw.Teams != nil && len(aw.Teams) > 0 {
			teams := make([]interface{}, len(aw.Teams))
			for i, v := range aw.Teams {
				teams[i] = v
			}
			if err := d.Set("authorized_writer_teams", schema.NewSet(schema.HashString, teams)); err != nil {
				return err
			}
		}
		if aw.Users != nil && len(aw.Users) > 0 {
			users := make([]interface{}, len(aw.Users))
			for i, v := range aw.Users {
				users[i] = v
			}
			if err := d.Set("authorized_writer_users", schema.NewSet(schema.HashString, users)); err != nil {
				return err
			}
		}
	}

	// The column and grid layouts are purely a terraform-side function and
	// the API has no awareness of it. See the documentation for the dashboard
	// resource for further discussion.
	defaultLayout := true
	if gridTF, ok := d.GetOk("grid"); ok {
		if gridList, tok := gridTF.([]interface{}); tok {
			if len(gridList) > 0 {
				defaultLayout = false
			}
		}
	} else if colTF, ok := d.GetOk("column"); ok {
		if colList, tok := colTF.([]interface{}); tok {
			if len(colList) > 0 {
				defaultLayout = false
			}
		}
	}

	if defaultLayout {
		charts := make([]map[string]interface{}, len(dash.Charts))
		for i, c := range dash.Charts {
			chart := make(map[string]interface{})
			chart["chart_id"] = c.ChartId
			chart["height"] = c.Height
			chart["width"] = c.Width
			chart["row"] = c.Row
			chart["column"] = c.Column
			charts[i] = chart
		}
		if err := d.Set("chart", charts); err != nil {
			return err
		}
	}

	// Filters
	if dash.Filters != nil {
		filters := dash.Filters
		// Map Sources to filters
		if len(filters.Sources) > 0 {
			tfFilters := make([]map[string]interface{}, len(filters.Sources))
			for i, source := range filters.Sources {
				tfFilter := make(map[string]interface{})
				tfFilter["negated"] = source.NOT
				tfFilter["property"] = source.Property
				tfFilter["values"] = flattenStringSliceToSet(source.Value)
				tfFilter["apply_if_exist"] = source.ApplyIfExists
				tfFilters[i] = tfFilter
			}
			if err := d.Set("filter", tfFilters); err != nil {
				return err
			}
		}
		// Map Time to fields
		if filters.Time != nil {
			timeFilter := filters.Time
			if strings.ToUpper(string(timeFilter.End)) == "NOW" {
				if err := d.Set("time_range", timeFilter.Start); err != nil {
					return err
				}
			} else {
				if timeFilter.Start != "" {
					start, err := strconv.Atoi(string(timeFilter.Start))
					if err != nil {
						return fmt.Errorf("Unable to convert start time %s to integer: %s", timeFilter.Start, err)
					}
					if err := d.Set("start_time", start/1000); err != nil {
						return err
					}
				}
				if timeFilter.End != "" {
					end, err := strconv.Atoi(string(timeFilter.End))
					if err != nil {
						return fmt.Errorf("Unable to convert end time %s to integer: %s", timeFilter.End, err)
					}
					if err := d.Set("end_time", end/1000); err != nil {
						return err
					}
				}
			}
		}
		// Map variables to variable
		if len(filters.Variables) > 0 {
			dashVars := make([]map[string]interface{}, len(filters.Variables))
			for i, v := range filters.Variables {
				dashVar := make(map[string]interface{})
				dashVar["property"] = v.Property
				dashVar["alias"] = v.Alias
				dashVar["description"] = v.Description
				dashVar["values"] = flattenStringSliceToSet(v.Value)
				dashVar["value_required"] = v.Required
				dashVar["values_suggested"] = flattenStringSliceToSet(v.PreferredSuggestions)
				dashVar["restricted_suggestions"] = v.Restricted
				dashVar["replace_only"] = v.ReplaceOnly
				dashVar["apply_if_exist"] = v.ApplyIfExists
				dashVars[i] = dashVar
			}
			if err := d.Set("variable", dashVars); err != nil {
				return err
			}
		}
	}

	// Chart Event Overlays
	if len(dash.EventOverlays) > 0 {
		evOverlays := make([]map[string]interface{}, len(dash.EventOverlays))
		for i, v := range dash.EventOverlays {
			evOverlay := make(map[string]interface{})
			evOverlay["line"] = v.EventLine
			evOverlay["label"] = v.Label

			if v.EventColorIndex != nil {
				colorName, err := getNameFromPaletteColorsByIndex(int(*v.EventColorIndex))
				if err != nil {
					return fmt.Errorf("Unknown event overlay color: %d", v.EventColorIndex)
				}
				evOverlay["color"] = colorName
			}
			if v.EventSignal != nil {
				evOverlay["signal"] = v.EventSignal.EventSearchText
				evOverlay["type"] = v.EventSignal.EventType
			}
			evOverlays[i] = evOverlay

			if len(v.Sources) > 0 {
				sources := make([]map[string]interface{}, len(v.Sources))
				for i, s := range v.Sources {
					source := make(map[string]interface{})
					source["negated"] = s.NOT
					source["values"] = flattenStringSliceToSet(s.Value)
					source["property"] = s.Property
					sources[i] = source
				}
				evOverlay["source"] = sources
			}
		}
		if err := d.Set("event_overlay", evOverlays); err != nil {
			return err
		}
	}

	// Chart Selected Event Overlays
	if len(dash.SelectedEventOverlays) > 0 {
		sevs := make([]map[string]interface{}, len(dash.SelectedEventOverlays))
		for i, s := range dash.SelectedEventOverlays {
			evOverlay := make(map[string]interface{})
			if s.EventSignal != nil {
				evOverlay["signal"] = s.EventSignal.EventSearchText
				evOverlay["type"] = s.EventSignal.EventType
			}
			sevs[i] = evOverlay

			if len(s.Sources) > 0 {
				sources := make([]map[string]interface{}, len(s.Sources))
				for i, s := range s.Sources {
					source := make(map[string]interface{})
					source["negated"] = s.NOT
					source["values"] = flattenStringSliceToSet(s.Value)
					source["property"] = s.Property
					sources[i] = source
				}
				evOverlay["source"] = sources
			}
		}
		if err := d.Set("selected_event_overlay", sevs); err != nil {
			return err
		}
	}

	if dash.DiscoveryOptions != nil {
		if err := d.Set("discovery_options_query", dash.DiscoveryOptions.Query); err != nil {
			return err
		}
		if err := d.Set("discovery_options_selectors", flattenStringSliceToSet(*dash.DiscoveryOptions.Selectors)); err != nil {
			return err
		}
	}

	return nil
}

func dashboardUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Dashboard Payload: %s", string(debugOutput))

	dash, err := config.Client.UpdateDashboard(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Dashboard Response: %v", dash)
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DashboardAppPath+dash.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(dash.Id)
	return dashboardAPIToTF(d, dash)
}

func dashboardDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	err := config.Client.DeleteDashboard(context.TODO(), d.Id())
	return err
}

/*
  Validate Chart Resolution option against a list of allowed words.
*/
func validateChartsResolution(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"default", "low", "high", "highest"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}

func validateEventOverlayType(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"eventTimeSeries", "detectorEvents"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}
