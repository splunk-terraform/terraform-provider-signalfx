package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/dashboard_group"
	"github.com/signalfx/signalfx-go/navigator"
)

const (
	NavigatorAppPath = "/#/infra/entity/"
)

func navigatorResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"navigator_code": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "",
			},
			"display_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "",
			},
			"id_display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"property_identifier_template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "",
			},
			"entity_metrics": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"display_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"value_label": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"value_format": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"metric_selectors": {
							Type:        schema.TypeList,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"job": {
							Type:        schema.TypeMap,
							Required:    true,
							Description: "",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resolution": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "",
									},
									"template": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "",
									},
									"var_name": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "",
									},
									"filters": {
										Type:        schema.TypeList,
										Required:    true,
										Description: "",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"property": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "",
												},
												"property_value": {
													Type:        schema.TypeList,
													Optional:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
													Description: "",
												},
												"not": {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "",
												},
												"type": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"instance_label": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"system_types": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			"import_qualifiers": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"filters": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"property": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "",
									},
									"values": {
										Type:        schema.TypeList,
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "",
									},
									"not": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "",
									},
								},
							},
						},
					},
				},
			},
			"category": {
				Type:        schema.TypeMap,
				Required:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"category_code": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"category_display_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"category_group_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"category_instance_label": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"connected_category": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "",
						},
					},
				},
			},
			"default_group_by": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"alert_query": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"list_columns": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"display_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"format": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"property": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"summary_metric_label": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"summary_metric_program_text": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"tooltip_key_list": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"display_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"format": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						"is_summary_property": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "",
						},
						"property": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"dashboard_discovery_query": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"dashboard_mts_query": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"required_properties": {
				Type:        schema.TypeList,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			"aggregate_dashboard_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"instance_dashboard_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"dashboard_name_match": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"aggregate_dashboards": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			"instance_dashboards": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			"instance_display_text": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the dashboard",
			},
		},
		Create: navigatorCreate,
		Read:   navigatorRead,
		Update: navigatorUpdate,
		Delete: navigatorDelete,
		Exists: navigatorExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
Use Resource object to construct json payload in order to create a navigator
*/
func getPayloadNavigator(d *schema.ResourceData) (*navigator.CreateUpdateNavigatorRequest, error) {
	cunr := &navigator.CreateUpdateNavigatorRequest{
		NavigatorCode:  d.Get("navigator_code").(string),
		EntityMetrics:  getEntityMetrics(d.Get("entity_metrics").([]interface{})),
		Category:       getCategory(d.Get("category").(map[string]interface{})),
		ListColumns:    getListColumns(d.Get("list_columns").([]interface{})),
		TooltipKeyList: getTooltipKeyList(d.Get("tooltip_key_list").([]interface{})),
	}

	if val, ok := d.GetOk("display_name"); ok {
		displayName := val.(string)
		cunr.DisplayName = displayName
	}

	if val, ok := d.GetOk("id_display_name"); ok {
		idDisplayName := val.(string)
		cunr.IdDisplayName = idDisplayName
	}

	if val, ok := d.GetOk("property_identifier_template"); ok {
		propertyIdentifierTemplate := val.(string)
		cunr.PropertyIdentifierTemplate = propertyIdentifierTemplate
	}

	if val, ok := d.GetOk("instance_label"); ok {
		instanceLabel := val.(string)
		cunr.InstanceLabel = instanceLabel
	}

	if val, ok := d.GetOk("default_group_by"); ok {
		defaultGroupBy := val.(string)
		cunr.DefaultGroupBy = defaultGroupBy
	}

	if val, ok := d.GetOk("alert_query"); ok {
		alertQuery := val.(string)
		cunr.AlertQuery = alertQuery
	}

	if val, ok := d.GetOk("summary_metric_label"); ok {
		summaryMetricLabel := val.(string)
		cunr.SummaryMetricLabel = summaryMetricLabel
	}

	if val, ok := d.GetOk("summary_metric_program_text"); ok {
		summaryMetricProgramText := val.(string)
		cunr.SummaryMetricProgramText = summaryMetricProgramText
	}

	if val, ok := d.GetOk("dashboard_mts_query"); ok {
		dashboardMtsQuery := val.(string)
		cunr.DashboardMtsQuery = dashboardMtsQuery
	}

	if val, ok := d.GetOk("aggregate_dashboard_name"); ok {
		aggregateDashboardName := val.(string)
		cunr.AggregateDashboardName = aggregateDashboardName
	}

	if val, ok := d.GetOk("instance_dashboard_name"); ok {
		instanceDashboardName := val.(string)
		cunr.InstanceDashboardName = instanceDashboardName
	}

	if val, ok := d.GetOk("dashboard_name_match"); ok {
		dashboardNameMatch := val.(string)
		cunr.DashboardNameMatch = dashboardNameMatch
	}

	if val, ok := d.GetOk("instance_display_text"); ok {
		instanceDisplayText := val.(string)
		cunr.InstanceDisplayText = instanceDisplayText
	}

	if val, ok := d.GetOk("system_types"); ok {
		var systemTypes []string
		tfSystemTypes := val.([]interface{})
		for _, v := range tfSystemTypes {
			systemTypes = append(systemTypes, v.(string))
		}
		cunr.SystemTypes = systemTypes
	}

	if val, ok := d.GetOk("dashboard_discovery_query"); ok {
		var dashboardDiscoveryQuery []string
		tfDashboardDiscoveryQuery := val.([]interface{})
		for _, v := range tfDashboardDiscoveryQuery {
			dashboardDiscoveryQuery = append(dashboardDiscoveryQuery, v.(string))
		}
		cunr.DashboardDiscoveryQuery = dashboardDiscoveryQuery
	}

	if val, ok := d.GetOk("required_properties"); ok {
		var requiredProperties []string
		tfRequiredProperties := val.([]interface{})
		for _, v := range tfRequiredProperties {
			requiredProperties = append(requiredProperties, v.(string))
		}
		cunr.RequiredProperties = requiredProperties
	}

	if val, ok := d.GetOk("aggregate_dashboards"); ok {
		var aggregateDashboards []string
		tfAggregateDashboards := val.([]interface{})
		for _, v := range tfAggregateDashboards {
			aggregateDashboards = append(aggregateDashboards, v.(string))
		}
		cunr.AggregateDashboards = aggregateDashboards
	}

	if val, ok := d.GetOk("instance_dashboards"); ok {
		var instanceDashboards []string
		tfInstanceDashboards := val.([]interface{})
		for _, v := range tfInstanceDashboards {
			instanceDashboards = append(instanceDashboards, v.(string))
		}
		cunr.InstanceDashboards = instanceDashboards
	}

	if tfiq, ok := d.GetOk("import_qualifier"); ok {
		tfIQs := tfiq.([]interface{})
		iqs := make([]*dashboard_group.ImportQualifier, len(tfIQs))
		for i, iq := range tfIQs {
			iq := iq.(map[string]interface{})

			var qualifierMetric string
			if val, ok := iq["metric"]; ok {
				qualifierMetric = val.(string)
			}

			filtersList := iq["filters"].([]interface{})
			filters := make([]*dashboard_group.ImportFilter, len(filtersList))
			for i, f := range filtersList {
				f := f.(map[string]interface{})

				var negated bool
				if val, ok := f["negated"]; ok {
					negated = val.(bool)
				}
				var property string
				if val, ok := f["property"]; ok {
					property = val.(string)
				}
				var values []string
				tfValues := f["values"].([]interface{})
				if len(tfValues) > 0 {
					values = []string{}
					for _, v := range tfValues {
						values = append(values, v.(string))
					}
				}

				filters[i] = &dashboard_group.ImportFilter{
					NOT:      negated,
					Property: property,
					Values:   values,
				}
			}

			iqs[i] = &dashboard_group.ImportQualifier{
				Metric:  qualifierMetric,
				Filters: filters,
			}
		}
		cunr.ImportQualifiers = iqs
	}

	return cunr, nil
}

func getEntityMetrics(set []interface{}) []*navigator.Metric {
	if len(set) > 0 {
		metricList := make([]*navigator.Metric, len(set))
		for i, metric := range set {
			metricData := metric.(map[string]interface{})
			var metricSelectors []string
			tfMetricSelectors := metricData["metric_selectors"].([]interface{})
			if len(tfMetricSelectors) > 0 {
				metricSelectors = []string{}
				for _, selector := range tfMetricSelectors {
					metricSelectors = append(metricSelectors, selector.(string))
				}
			}

			var metricId string
			if val, ok := metricData["id"]; ok {
				metricId = val.(string)
			}
			var metricType string
			if val, ok := metricData["type"]; ok {
				metricType = val.(string)
			}
			var displayName string
			if val, ok := metricData["display_name"]; ok {
				displayName = val.(string)
			}
			var valueLabel string
			if val, ok := metricData["value_label"]; ok {
				valueLabel = val.(string)
			}
			var valueFormat string
			if val, ok := metricData["value_format"]; ok {
				valueFormat = val.(string)
			}
			var description string
			if val, ok := metricData["description"]; ok {
				description = val.(string)
			}

			metricList[i] = &navigator.Metric{
				Id:              metricId,
				Type:            metricType,
				DisplayName:     displayName,
				ValueLabel:      valueLabel,
				ValueFormat:     valueFormat,
				MetricSelectors: metricSelectors,
				Description:     description,
				Job:             getMetricJob(metricData["job"].(map[string]interface{})),
				ColoringScheme: &navigator.ColoringScheme{
					Palette:  "GREEN_RED",
					MaxValue: 1,
				},
			}
		}
		return metricList
	}
	return nil
}

func getMetricJob(tfJob map[string]interface{}) *navigator.Job {
	var filters []*navigator.JobFilter
	if tfFilters, ok := tfJob["filters"]; ok {
		tfFilters := tfFilters.([]interface{})
		if len(tfFilters) > 0 {
			filters := make([]*navigator.JobFilter, len(tfFilters))
			for i, f := range tfFilters {
				filter := f.(map[string]interface{})
				var property string
				if val, ok := filter["property"]; ok {
					property = val.(string)
				}
				var values []string
				tfValues := filter["values"].([]interface{})
				if len(tfValues) > 0 {
					values = []string{}
					for _, v := range tfValues {
						values = append(values, v.(string))
					}
				}
				var negated bool
				if val, ok := filter["property"]; ok {
					negated = val.(bool)
				}
				var filterType string
				if val, ok := filter["property"]; ok {
					filterType = val.(string)
				}

				filters[i] = &navigator.JobFilter{
					Property:      property,
					PropertyValue: values,
					Not:           negated,
					Type:          filterType,
				}
			}
		}
	}

	// Commented out as currently getting error on receiving resource
	// var resolution int32
	// if val, ok := tfJob["resolution"]; ok {
	// 	resolutionStr := val.(string)
	// 	if resolutionInt, err := strconv.ParseInt(resolutionStr, 10, 16); err == nil {
	// 		resolution = int32(resolutionInt)
	// 	}
	// }

	var template string
	if val, ok := tfJob["template"]; ok {
		template = val.(string)
	}
	var varName string
	if val, ok := tfJob["var_name"]; ok {
		varName = val.(string)
	}

	return &navigator.Job{
		// Resolution: resolution,
		Resolution: 600000,
		Template:   template,
		VarName:    varName,
		Filters:    filters,
	}
}

func getCategory(tfCategory map[string]interface{}) *navigator.Category {
	var categoryCode string
	if val, ok := tfCategory["category_code"]; ok {
		categoryCode = val.(string)
	}
	var categoryDisplayName string
	if val, ok := tfCategory["category_display_name"]; ok {
		categoryDisplayName = val.(string)
	}
	var categoryGroupName string
	if val, ok := tfCategory["category_group_name"]; ok {
		categoryGroupName = val.(string)
	}
	var categoryInstanceLabel string
	if val, ok := tfCategory["category_instance_label"]; ok {
		categoryInstanceLabel = val.(string)
	}
	var connectedCategory bool
	if val, ok := tfCategory["connected_category"]; ok {
		connectedCategory = val.(bool)
	}

	return &navigator.Category{
		CategoryCode:          categoryCode,
		CategoryDisplayName:   categoryDisplayName,
		CategoryGroupName:     categoryGroupName,
		CategoryInstanceLabel: categoryInstanceLabel,
		ConnectedCategory:     connectedCategory,
	}
}

func getListColumns(set []interface{}) []*navigator.ListColumn {
	if len(set) > 0 {
		listColumns := make([]*navigator.ListColumn, len(set))
		for i, column := range set {
			listColumnData := column.(map[string]interface{})
			var displayName string
			if val, ok := listColumnData["display_name"]; ok {
				displayName = val.(string)
			}
			var format string
			if val, ok := listColumnData["format"]; ok {
				format = val.(string)
			}
			var property string
			if val, ok := listColumnData["property"]; ok {
				property = val.(string)
			}
			listColumns[i] = &navigator.ListColumn{
				DisplayName: displayName,
				Format:      format,
				Property:    property,
			}
		}
		return listColumns
	}
	return nil
}

func getTooltipKeyList(set []interface{}) []*navigator.TooltipKey {
	if len(set) > 0 {
		tooltipKeys := make([]*navigator.TooltipKey, len(set))
		for i, tooltipKey := range set {
			tooltipKeyData := tooltipKey.(map[string]interface{})
			var displayName string
			if val, ok := tooltipKeyData["display_name"]; ok {
				displayName = val.(string)
			}
			var format string
			if val, ok := tooltipKeyData["format"]; ok {
				format = val.(string)
			}
			var isSummaryProperty bool
			if val, ok := tooltipKeyData["is_summary_property"]; ok {
				isSummaryProperty = val.(bool)
			}
			var property string
			if val, ok := tooltipKeyData["property"]; ok {
				property = val.(string)
			}
			tooltipKeys[i] = &navigator.TooltipKey{
				DisplayName:       displayName,
				Format:            format,
				IsSummaryProperty: isSummaryProperty,
				Property:          property,
			}
		}
		return tooltipKeys
	}
	return nil
}

func navigatorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadNavigator(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Navigator Create Payload: %s", debugOutput)

	nav, err := config.Client.CreateNavigator(context.TODO(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL := config.CustomAppURL + NavigatorAppPath + nav.NavigatorCode
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(nav.Id)

	return navigatorAPIToTF(d, nav)
}

func navigatorExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetNavigator(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func navigatorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	nav, err := config.Client.GetNavigator(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	appURL := config.CustomAppURL + NavigatorAppPath + nav.NavigatorCode
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}

	return navigatorAPIToTF(d, nav)
}

func navigatorAPIToTF(d *schema.ResourceData, nav *navigator.Navigator) error {
	debugOutput, _ := json.Marshal(nav)
	log.Printf("[DEBUG] SignalFx: Got Navigator to enState: %s", string(debugOutput))

	if err := d.Set("navigator_code", nav.NavigatorCode); err != nil {
		return err
	}
	if err := d.Set("display_name", nav.DisplayName); err != nil {
		return err
	}
	if err := d.Set("id_display_name", nav.IdDisplayName); err != nil {
		return err
	}
	if err := d.Set("property_identifier_template", nav.PropertyIdentifierTemplate); err != nil {
		return err
	}

	entityMetrics := make([]map[string]interface{}, len(nav.EntityMetrics))
	for i, m := range nav.EntityMetrics {
		metric := make(map[string]interface{})
		metric["id"] = m.Id
		metric["type"] = m.Type
		metric["display_name"] = m.DisplayName
		metric["value_label"] = m.ValueLabel
		metric["value_format"] = m.ValueFormat

		if len(m.MetricSelectors) > 0 {
			selectors := make([]string, len(m.MetricSelectors))
			for i, s := range m.MetricSelectors {
				selectors[i] = s
			}
			metric["metric_selectors"] = selectors
		}
		metric["description"] = m.Description

		job := make(map[string]interface{})
		// job["resolution"] = m.Job.Resolution
		job["template"] = m.Job.Template
		job["var_name"] = m.Job.VarName
		if len(m.Job.Filters) > 0 {
			jobFilters := make([]map[string]interface{}, len(m.Job.Filters))
			for i, jf := range m.Job.Filters {
				filter := make(map[string]interface{})
				filter["property"] = jf.Property
				filter["property_value"] = jf.PropertyValue
				filter["not"] = jf.Not
				filter["type"] = jf.Type
				jobFilters[i] = filter
			}
			job["filters"] = jobFilters
		}
		metric["job"] = job
		entityMetrics[i] = metric
	}
	if err := d.Set("entity_metrics", entityMetrics); err != nil {
		return err
	}

	if err := d.Set("instance_label", nav.InstanceLabel); err != nil {
		return err
	}

	if len(nav.SystemTypes) > 0 {
		systemTypes := make([]string, len(nav.SystemTypes))
		for i, s := range nav.SystemTypes {
			systemTypes[i] = s
		}
		if err := d.Set("system_types", systemTypes); err != nil {
			return err
		}
	}

	if len(nav.ImportQualifiers) > 0 {
		qualifiers := make([]map[string]interface{}, len(nav.ImportQualifiers))
		for i, qualifier := range nav.ImportQualifiers {
			iq := make(map[string]interface{})
			iq["metric"] = qualifier.Metric
			filters := make([]map[string]interface{}, len(qualifier.Filters))
			for j, apiFilter := range qualifier.Filters {
				filter := make(map[string]interface{})
				filter["negated"] = apiFilter.NOT
				filter["property"] = apiFilter.Property
				filter["values"] = flattenStringSliceToSet(apiFilter.Values)
				filters[j] = filter
			}
			iq["filters"] = filters
			qualifiers[i] = iq
		}
		if err := d.Set("import_qualifier", qualifiers); err != nil {
			return err
		}
	}

	category := make(map[string]interface{})
	category["category_code"] = nav.Category.CategoryCode
	category["category_display_name"] = nav.Category.CategoryDisplayName
	category["category_group_name"] = nav.Category.CategoryGroupName
	category["category_instance_label"] = nav.Category.CategoryInstanceLabel

	// Commented out as currently getting error on receiving resource
	// category["connected_category"] = nav.Category.ConnectedCategory

	if err := d.Set("category", category); err != nil {
		return err
	}

	if err := d.Set("default_group_by", nav.DefaultGroupBy); err != nil {
		return err
	}
	if err := d.Set("alert_query", nav.AlertQuery); err != nil {
		return err
	}

	if len(nav.ListColumns) > 0 {
		columns := make([]map[string]interface{}, len(nav.ListColumns))
		for i, listColumn := range nav.ListColumns {
			lc := make(map[string]interface{})
			lc["display_name"] = listColumn.DisplayName
			lc["format"] = listColumn.Format
			lc["property"] = listColumn.Property
			columns[i] = lc
		}
		if err := d.Set("list_columns", columns); err != nil {
			return err
		}
	}

	if err := d.Set("summary_metric_label", nav.SummaryMetricLabel); err != nil {
		return err
	}
	if err := d.Set("summary_metric_program_text", nav.SummaryMetricProgramText); err != nil {
		return err
	}

	if len(nav.TooltipKeyList) > 0 {
		keyList := make([]map[string]interface{}, len(nav.TooltipKeyList))
		for i, key := range nav.TooltipKeyList {
			tk := make(map[string]interface{})
			tk["display_name"] = key.DisplayName
			tk["format"] = key.Format
			tk["is_summary_property"] = key.IsSummaryProperty
			tk["property"] = key.Property
			keyList[i] = tk
		}
		if err := d.Set("tooltip_key_list", keyList); err != nil {
			return err
		}
	}

	if len(nav.DashboardDiscoveryQuery) > 0 {
		dashboardDiscoveryQuery := make([]string, len(nav.DashboardDiscoveryQuery))
		for i, q := range nav.DashboardDiscoveryQuery {
			dashboardDiscoveryQuery[i] = q
		}
		if err := d.Set("dashboard_discovery_query", dashboardDiscoveryQuery); err != nil {
			return err
		}
	}

	if err := d.Set("dashboard_mts_query", nav.DashboardMtsQuery); err != nil {
		return err
	}

	if len(nav.RequiredProperties) > 0 {
		requiredProperties := make([]string, len(nav.RequiredProperties))
		for i, p := range nav.RequiredProperties {
			requiredProperties[i] = p
		}
		if err := d.Set("required_properties", requiredProperties); err != nil {
			return err
		}
	}

	if err := d.Set("aggregate_dashboard_name", nav.AggregateDashboardName); err != nil {
		return err
	}
	if err := d.Set("instance_dashboard_name", nav.InstanceDashboardName); err != nil {
		return err
	}
	if err := d.Set("dashboard_name_match", nav.DashboardNameMatch); err != nil {
		return err
	}

	if len(nav.AggregateDashboards) > 0 {
		aggregateDashboards := make([]string, len(nav.AggregateDashboards))
		for i, dashboard := range nav.AggregateDashboards {
			aggregateDashboards[i] = dashboard
		}
		if err := d.Set("aggregate_dashboards", aggregateDashboards); err != nil {
			return err
		}
	}
	if len(nav.InstanceDashboards) > 0 {
		instanceDashboards := make([]string, len(nav.InstanceDashboards))
		for i, dashboard := range nav.InstanceDashboards {
			instanceDashboards[i] = dashboard
		}
		if err := d.Set("instance_dashboards", instanceDashboards); err != nil {
			return err
		}
	}

	if err := d.Set("instance_display_text", nav.InstanceDisplayText); err != nil {
		return err
	}

	return nil
}

func navigatorUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadNavigator(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Navigator Payload: %s", string(debugOutput))

	nav, err := config.Client.UpdateNavigator(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Navigator Response: %v", nav)
	// Since things worked, set the URL and move on
	appURL := config.CustomAppURL + NavigatorAppPath + nav.NavigatorCode
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(nav.Id)
	return navigatorAPIToTF(d, nav)
}

func navigatorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	err := config.Client.DeleteNavigator(context.TODO(), d.Id())
	return err
}
