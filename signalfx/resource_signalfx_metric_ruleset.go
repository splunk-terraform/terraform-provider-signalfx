// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go/metric_ruleset"
)

func metricRulesetResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metric_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the metric",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Version of the ruleset",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Information about the metric ruleset.",
			},
			"aggregation_rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Aggregation rules in the ruleset",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of this aggregation rule",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Information about an aggregation rule.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Status of this aggregation rule",
						},
						"matcher": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "The matcher for this rule",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "The type of the matcher",
										ValidateFunc: validation.StringInSlice([]string{"dimension"}, false),
									},
									"filters": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of filters to match on",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"not": {
													Type:        schema.TypeBool,
													Required:    true,
													Description: "Flag specifying equals or not equals",
												},
												"property": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of dimension to match",
												},
												"property_value": {
													Type:        schema.TypeSet,
													Required:    true,
													Description: "List of property values to match",
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
								},
							},
						},
						"aggregator": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "The aggregator for this rule",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "The type of the aggregator",
										ValidateFunc: validation.StringInSlice([]string{"rollup"}, false),
									},
									"output_name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The aggregated metric name",
									},
									"dimensions": {
										Type:        schema.TypeSet,
										Required:    true,
										Description: "List of dimensions to keep or drop in aggregated metric",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"drop_dimensions": {
										Type:        schema.TypeBool,
										Required:    true,
										Description: "Flag specifying to keep or drop given dimensions",
									},
								},
							},
						},
					},
				},
			},
			"exception_rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Exception rules in the ruleset",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of this exception rule",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Information about an exception rule.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Status of this exception rule",
						},
						"matcher": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "The matcher for this rule",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "The type of the matcher",
										ValidateFunc: validation.StringInSlice([]string{"dimension"}, false),
									},
									"filters": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of filters to match on",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"not": {
													Type:        schema.TypeBool,
													Required:    true,
													Description: "Flag specifying equals or not equals",
												},
												"property": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of dimension to match",
												},
												"property_value": {
													Type:        schema.TypeSet,
													Required:    true,
													Description: "List of dimension values to match",
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
								},
							},
						},
						"restoration": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Restoration for this rule",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"restoration_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "ID of the restoration job.",
									},
									"start_time": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Time from which the restoration job will restore archived data, in the form of *nix time in milliseconds.",
									},
									"stop_time": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Time to which the restoration job will restore archived data, in the form of *nix time in milliseconds.",
									},
								},
							},
						},
					},
				},
			},
			"routing_rule": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Location to send the input metric",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Destination to send the input metric",
							ValidateFunc: validation.StringInSlice([]string{"RealTime", "Archived", "Drop"}, false),
						},
					},
				},
			},
			"creator": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the creator of the metric ruleset",
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of when the metric ruleset was created",
			},
			"last_updated_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of user who last updated the metric ruleset",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of when the metric ruleset was last updated",
			},
			"last_updated_by_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of user who last updated this metric ruleset",
			},
		},

		Create: metricRulesetCreate,
		Read:   metricRulesetRead,
		Update: metricRulesetUpdate,
		Delete: metricRulesetDelete,
		Exists: metricRulesetExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func metricRulesetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payloadReq, err := getPayloadMetricRuleset(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	payload := metric_ruleset.CreateMetricRulesetRequest{
		AggregationRules: payloadReq.AggregationRules,
		ExceptionRules:   payloadReq.ExceptionRules,
		MetricName:       *payloadReq.MetricName,
		Description:      payloadReq.Description,
		RoutingRule:      *payloadReq.RoutingRule,
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Metric Ruleset Create Payload: %s", debugOutput)

	metricRulesetResp, err := config.Client.CreateMetricRuleset(context.TODO(), &payload)
	if err != nil {
		return err
	}

	metricRuleset := metric_ruleset.MetricRuleset{
		Id:                metricRulesetResp.Id,
		Version:           metricRulesetResp.Version,
		MetricName:        metricRulesetResp.MetricName,
		AggregationRules:  metricRulesetResp.AggregationRules,
		ExceptionRules:    metricRulesetResp.ExceptionRules,
		Description:       metricRulesetResp.Description,
		RoutingRule:       metricRulesetResp.RoutingRule,
		Creator:           metricRulesetResp.Creator,
		Created:           metricRulesetResp.Created,
		LastUpdated:       metricRulesetResp.LastUpdated,
		LastUpdatedBy:     metricRulesetResp.LastUpdatedBy,
		LastUpdatedByName: metricRulesetResp.LastUpdatedByName,
	}
	return metricRulesetAPIToTF(d, &metricRuleset)
}

func metricRulesetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	metricRulesetResp, err := config.Client.GetMetricRuleset(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	metricRuleset := metric_ruleset.MetricRuleset{
		Id:                metricRulesetResp.Id,
		Version:           metricRulesetResp.Version,
		MetricName:        metricRulesetResp.MetricName,
		Description:       metricRulesetResp.Description,
		AggregationRules:  metricRulesetResp.AggregationRules,
		ExceptionRules:    metricRulesetResp.ExceptionRules,
		RoutingRule:       metricRulesetResp.RoutingRule,
		Creator:           metricRulesetResp.Creator,
		Created:           metricRulesetResp.Created,
		LastUpdated:       metricRulesetResp.LastUpdated,
		LastUpdatedBy:     metricRulesetResp.LastUpdatedBy,
		LastUpdatedByName: metricRulesetResp.LastUpdatedByName,
	}

	return metricRulesetAPIToTF(d, &metricRuleset)
}

func metricRulesetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	currentMetricRuleset, err := config.Client.GetMetricRuleset(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	payloadReq, err := getPayloadMetricRuleset(d)
	payload := metric_ruleset.UpdateMetricRulesetRequest{
		AggregationRules: payloadReq.AggregationRules,
		ExceptionRules:   payloadReq.ExceptionRules,
		MetricName:       payloadReq.MetricName,
		Description:      payloadReq.Description,
		RoutingRule:      payloadReq.RoutingRule,
		Version:          currentMetricRuleset.Version,
	}
	if err != nil {
		return fmt.Errorf("Failed updating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Metric Ruleset Update Payload: %s", debugOutput)

	metricRulesetResp, err := config.Client.UpdateMetricRuleset(context.TODO(), d.Id(), &payload)
	if err != nil {
		return err
	}

	metricRuleset := metric_ruleset.MetricRuleset{
		AggregationRules:  metricRulesetResp.AggregationRules,
		ExceptionRules:    metricRulesetResp.ExceptionRules,
		Creator:           metricRulesetResp.Creator,
		CreatorName:       metricRulesetResp.CreatorName,
		Created:           metricRulesetResp.Created,
		Id:                metricRulesetResp.Id,
		LastUpdatedBy:     metricRulesetResp.LastUpdatedBy,
		LastUpdatedByName: metricRulesetResp.LastUpdatedByName,
		LastUpdated:       metricRulesetResp.LastUpdated,
		MetricName:        metricRulesetResp.MetricName,
		Description:       metricRulesetResp.Description,
		RoutingRule:       metricRulesetResp.RoutingRule,
		Version:           metricRulesetResp.Version,
	}

	return metricRulesetAPIToTF(d, &metricRuleset)
}

func metricRulesetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	err := config.Client.DeleteMetricRuleset(context.TODO(), d.Id())
	return err
}

func metricRulesetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetMetricRuleset(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func metricRulesetAPIToTF(d *schema.ResourceData, metricRuleset *metric_ruleset.MetricRuleset) error {
	debugOutput, _ := json.Marshal(metricRuleset)
	log.Printf("[DEBUG] SignalFx: Got MetricRuleset to enState: %s", string(debugOutput))

	d.SetId(*metricRuleset.Id)
	if err := d.Set("metric_name", metricRuleset.MetricName); err != nil {
		return err
	}
	if err := d.Set("description", metricRuleset.Description); err != nil {
		return err
	}

	versionStr := strconv.FormatInt(*metricRuleset.Version, 10)
	if err := d.Set("version", versionStr); err != nil {
		return err
	}
	if err := d.Set("creator", metricRuleset.Creator); err != nil {
		return err
	}
	createdStr := strconv.FormatInt(*metricRuleset.Created, 10)
	if err := d.Set("created", createdStr); err != nil {
		return err
	}
	if err := d.Set("last_updated_by", metricRuleset.LastUpdatedBy); err != nil {
		return err
	}
	lastUpdatedStr := strconv.FormatInt(*metricRuleset.LastUpdated, 10)
	if err := d.Set("last_updated", lastUpdatedStr); err != nil {
		return err
	}

	if metricRuleset.AggregationRules != nil {
		rules := make([]map[string]interface{}, len(metricRuleset.AggregationRules))
		for i, rule := range metricRuleset.AggregationRules {
			aggRule := map[string]interface{}{
				"name":        rule.Name,
				"enabled":     rule.Enabled,
				"description": rule.Description,
			}

			filters := make([]map[string]interface{}, len(rule.Matcher.DimensionMatcher.Filters))
			for j, filter := range rule.Matcher.DimensionMatcher.Filters {
				entry := map[string]interface{}{
					"property":       filter.Property,
					"property_value": filter.PropertyValue,
					"not":            *filter.NOT,
				}
				filters[j] = entry
			}

			matcher := map[string]interface{}{
				"type":    rule.Matcher.DimensionMatcher.Type,
				"filters": filters,
			}
			aggRule["matcher"] = []map[string]interface{}{matcher}

			dimensions := make([]interface{}, len(rule.Aggregator.RollupAggregator.Dimensions))
			for j, dim := range rule.Aggregator.RollupAggregator.Dimensions {
				dimensions[j] = dim
			}
			aggregator := map[string]interface{}{
				"type":            rule.Aggregator.RollupAggregator.Type,
				"output_name":     rule.Aggregator.RollupAggregator.OutputName,
				"drop_dimensions": *rule.Aggregator.RollupAggregator.DropDimensions,
				"dimensions":      dimensions,
			}
			aggRule["aggregator"] = []map[string]interface{}{aggregator}

			rules[i] = aggRule
		}
		if err := d.Set("aggregation_rules", rules); err != nil {
			return err
		}
	}

	if metricRuleset.ExceptionRules != nil {
		rules := make([]map[string]interface{}, len(metricRuleset.ExceptionRules))
		for i, rule := range metricRuleset.ExceptionRules {
			excRule := map[string]interface{}{
				"name":        rule.Name,
				"enabled":     rule.Enabled,
				"description": rule.Description,
			}

			filters := make([]map[string]interface{}, len(rule.Matcher.Filters))
			for j, filter := range rule.Matcher.Filters {
				entry := map[string]interface{}{
					"property":       filter.Property,
					"property_value": filter.PropertyValue,
					"not":            *filter.NOT,
				}
				filters[j] = entry
			}

			matcher := map[string]interface{}{
				"type":    rule.Matcher.Type,
				"filters": filters,
			}
			excRule["matcher"] = []map[string]interface{}{matcher}

			if val, ok := rule.GetRestorationOk(); ok {
				if val != nil && val.StartTime != nil && *val.StartTime > int64(0) && val.StopTime != nil && *val.StopTime > int64(0) {
					restoration := map[string]interface{}{
						"restoration_id": *val.RestorationId,
						"start_time":     strconv.FormatInt(*val.StartTime, 10),
						"stop_time":      strconv.FormatInt(*val.StopTime, 10),
					}
					excRule["restoration"] = []map[string]interface{}{restoration}
				}
			}

			rules[i] = excRule
		}
		if err := d.Set("exception_rules", rules); err != nil {
			return err
		}
	}

	dest := map[string]interface{}{"destination": metricRuleset.RoutingRule.Destination}
	routingRule := []interface{}{dest}
	if err := d.Set("routing_rule", routingRule); err != nil {
		return err
	}

	return nil
}

func getPayloadMetricRuleset(d *schema.ResourceData) (*metric_ruleset.MetricRuleset, error) {
	metricName := d.Get("metric_name").(string)
	var description = ""
	if val, ok := d.GetOk("description"); ok {
		description = val.(string)
	}
	cudr := &metric_ruleset.MetricRuleset{
		MetricName:       &metricName,
		Description:      &description,
		AggregationRules: []metric_ruleset.AggregationRule{},
		ExceptionRules:   []metric_ruleset.ExceptionRule{},
		RoutingRule:      &metric_ruleset.RoutingRule{},
	}

	if val, ok := d.Get("aggregation_rules").([]interface{}); ok {
		cudr.AggregationRules = getAggregationRules(val)
	}

	if val, ok := d.Get("exception_rules").([]interface{}); ok {
		cudr.ExceptionRules = getExceptionRules(val)
	}

	if val, ok := d.GetOk("routing_rule"); ok && len(val.(*schema.Set).List()) > 0 {
		routingRule := val.(*schema.Set).List()[0].(map[string]interface{})
		rr := getRoutingRule(routingRule)
		cudr.RoutingRule = &rr
	}

	return cudr, nil
}

func getAggregationRules(tfRules []interface{}) []metric_ruleset.AggregationRule {
	var aggregationRulesList []metric_ruleset.AggregationRule
	for _, tfRule := range tfRules {
		newTfRule := tfRule.(map[string]interface{})
		ruleName := newTfRule["name"].(string)
		var description = ""
		if val, ok := newTfRule["description"]; ok {
			description = val.(string)
		}
		rule := metric_ruleset.AggregationRule{
			Name:        &ruleName,
			Description: &description,
			Enabled:     newTfRule["enabled"].(bool),
			Matcher:     getMatcher(newTfRule),
			Aggregator:  getAggregator(newTfRule),
		}
		aggregationRulesList = append(aggregationRulesList, rule)
	}

	return aggregationRulesList
}

func getExceptionRules(tfRules []interface{}) []metric_ruleset.ExceptionRule {
	var exceptionRulesList []metric_ruleset.ExceptionRule
	for _, tfRule := range tfRules {
		newTfRule := tfRule.(map[string]interface{})
		var description = ""
		if val, ok := newTfRule["description"]; ok {
			description = val.(string)
		}
		rule := metric_ruleset.ExceptionRule{
			Name:        newTfRule["name"].(string),
			Description: &description,
			Enabled:     newTfRule["enabled"].(bool),
			Matcher:     getDimensionMatcher(newTfRule),
			Restoration: &metric_ruleset.ExceptionRuleRestorationFields{},
		}
		restFields := getRestoration(newTfRule)
		rule.Restoration = &restFields
		exceptionRulesList = append(exceptionRulesList, rule)
	}

	return exceptionRulesList
}

func getRestoration(tfRule map[string]interface{}) metric_ruleset.ExceptionRuleRestorationFields {
	if tfRule["restoration"] != nil && len(tfRule["restoration"].(*schema.Set).List()) > 0 {
		restoration := tfRule["restoration"].(*schema.Set).List()[0].(map[string]interface{})

		var restorationId = ""
		if val, ok := restoration["restoration_id"]; ok {
			restorationId = val.(string)
		} else {
			log.Printf("[DEBUG] SignalFx: restoration_id does not exist.")
		}
		startTime, err := strconv.ParseInt(restoration["start_time"].(string), 10, 64)
		if err != nil {
			panic(err)
		}

		restorationFields := &metric_ruleset.ExceptionRuleRestorationFields{
			RestorationId: &restorationId,
			StartTime:     &startTime,
		}

		val := restoration["stop_time"]
		if val != nil && len(val.(string)) > 0 {
			stopTime, err1 := strconv.ParseInt(val.(string), 10, 64)
			if err1 != nil {
				panic(err1)
			}
			restorationFields.StopTime = &stopTime
		}

		return *restorationFields
	} else {
		restorationFields := &metric_ruleset.ExceptionRuleRestorationFields{}
		return *restorationFields
	}

}

func getMatcher(tfRule map[string]interface{}) metric_ruleset.MetricMatcher {
	matcher := tfRule["matcher"].(*schema.Set).List()[0].(map[string]interface{})
	filters := make([]interface{}, 0)
	if matcher["filters"] != nil {
		filters = matcher["filters"].([]interface{})
	}

	metricMatcher := metric_ruleset.MetricMatcher{
		DimensionMatcher: &metric_ruleset.DimensionMatcher{
			Type:    matcher["type"].(string),
			Filters: getFilters(filters),
		},
	}

	return metricMatcher
}

func getDimensionMatcher(tfRule map[string]interface{}) metric_ruleset.DimensionMatcher {
	matcher := tfRule["matcher"].(*schema.Set).List()[0].(map[string]interface{})
	filters := make([]interface{}, 0)
	if matcher["filters"] != nil {
		filters = matcher["filters"].([]interface{})
	}

	dimensionMatcher := metric_ruleset.DimensionMatcher{
		Type:    matcher["type"].(string),
		Filters: getFilters(filters),
	}

	return dimensionMatcher
}

func getFilters(filters []interface{}) []metric_ruleset.PropertyFilter {
	var filterList []metric_ruleset.PropertyFilter
	for _, filter := range filters {
		filter := filter.(map[string]interface{})
		property := filter["property"].(string)
		not := filter["not"].(bool)
		var filterVals []string
		tfFilter := filter["property_value"].(*schema.Set).List()
		for _, tfFilter := range tfFilter {
			filterVals = append(filterVals, tfFilter.(string))
		}
		propFilter := metric_ruleset.PropertyFilter{
			Property:      &property,
			NOT:           &not,
			PropertyValue: filterVals,
		}
		filterList = append(filterList, propFilter)
	}
	return filterList
}

func getAggregator(tfRule map[string]interface{}) metric_ruleset.MetricAggregator {
	aggregator := tfRule["aggregator"].(*schema.Set).List()[0].(map[string]interface{})
	dropDimensions := aggregator["drop_dimensions"].(bool)

	var dimensions []string
	tfDims := aggregator["dimensions"].(*schema.Set).List()
	for _, tfDim := range tfDims {
		dimensions = append(dimensions, tfDim.(string))
	}

	metricAggregator := metric_ruleset.MetricAggregator{
		RollupAggregator: &metric_ruleset.RollupAggregator{
			Dimensions:     dimensions,
			DropDimensions: &dropDimensions,
			OutputName:     aggregator["output_name"].(string),
			Type:           aggregator["type"].(string),
		},
	}

	return metricAggregator
}

func getRoutingRule(routingRule map[string]interface{}) metric_ruleset.RoutingRule {
	destination := routingRule["destination"].(string)
	return metric_ruleset.RoutingRule{Destination: &destination}
}
