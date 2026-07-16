// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metricruleset "github.com/signalfx/signalfx-go/metric_ruleset"
)

type resourceMetricRulesetModel struct {
	ID                types.String `tfsdk:"id"`
	MetricName        types.String `tfsdk:"metric_name"`
	Version           types.String `tfsdk:"version"`
	Description       types.String `tfsdk:"description"`
	AggregationRules  types.List   `tfsdk:"aggregation_rules"`
	ExceptionRules    types.List   `tfsdk:"exception_rules"`
	RoutingRule       types.Object `tfsdk:"routing_rule"`
	Creator           types.String `tfsdk:"creator"`
	Created           types.Int64  `tfsdk:"created"`
	LastUpdatedBy     types.String `tfsdk:"last_updated_by"`
	LastUpdated       types.Int64  `tfsdk:"last_updated"`
	LastUpdatedByName types.String `tfsdk:"last_updated_by_name"`
}

type metricRulesetAggregationRuleModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Matcher     types.Object `tfsdk:"matcher"`
	Aggregator  types.Object `tfsdk:"aggregator"`
}

type metricRulesetExceptionRuleModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Matcher     types.Object `tfsdk:"matcher"`
	Restoration types.Object `tfsdk:"restoration"`
}

type metricRulesetMatcherModel struct {
	Type    types.String `tfsdk:"type"`
	Filters types.List   `tfsdk:"filters"`
}

type metricRulesetFilterModel struct {
	Not           types.Bool   `tfsdk:"not"`
	Property      types.String `tfsdk:"property"`
	PropertyValue types.Set    `tfsdk:"property_value"`
}

type metricRulesetAggregatorModel struct {
	Type           types.String `tfsdk:"type"`
	OutputName     types.String `tfsdk:"output_name"`
	Dimensions     types.Set    `tfsdk:"dimensions"`
	DropDimensions types.Bool   `tfsdk:"drop_dimensions"`
}

type metricRulesetRestorationModel struct {
	RestorationID types.String `tfsdk:"restoration_id"`
	StartTime     types.Int64  `tfsdk:"start_time"`
	StopTime      types.Int64  `tfsdk:"stop_time"`
}

type metricRulesetRoutingRuleModel struct {
	Destination types.String `tfsdk:"destination"`
}

type metricRulesetResponse interface {
	GetAggregationRulesOk() ([]metricruleset.AggregationRule, bool)
	GetCreatorOk() (*string, bool)
	GetCreatedOk() (*int64, bool)
	GetExceptionRulesOk() ([]metricruleset.ExceptionRule, bool)
	GetIdOk() (*string, bool)
	GetLastUpdatedByOk() (*string, bool)
	GetLastUpdatedByNameOk() (*string, bool)
	GetLastUpdatedOk() (*int64, bool)
	GetMetricNameOk() (*string, bool)
	GetDescriptionOk() (*string, bool)
	GetRoutingRuleOk() (*metricruleset.RoutingRule, bool)
	GetVersionOk() (*int64, bool)
}

type metricRulesetPayload struct {
	AggregationRules []metricruleset.AggregationRule
	ExceptionRules   []metricruleset.ExceptionRule
	MetricName       string
	Description      *string
	RoutingRule      metricruleset.RoutingRule
}

func (model resourceMetricRulesetModel) createRequest(ctx context.Context) (*metricruleset.CreateMetricRulesetRequest, diag.Diagnostics) {
	payload, diagnostics := model.payload(ctx)
	return &metricruleset.CreateMetricRulesetRequest{
		AggregationRules: payload.AggregationRules,
		ExceptionRules:   payload.ExceptionRules,
		MetricName:       payload.MetricName,
		Description:      payload.Description,
		RoutingRule:      payload.RoutingRule,
	}, diagnostics
}

func (model resourceMetricRulesetModel) updateRequest(ctx context.Context, version *int64) (*metricruleset.UpdateMetricRulesetRequest, diag.Diagnostics) {
	payload, diagnostics := model.payload(ctx)
	return &metricruleset.UpdateMetricRulesetRequest{
		AggregationRules: payload.AggregationRules,
		ExceptionRules:   payload.ExceptionRules,
		MetricName:       &payload.MetricName,
		Description:      payload.Description,
		RoutingRule:      &payload.RoutingRule,
		Version:          version,
	}, diagnostics
}

func (model resourceMetricRulesetModel) payload(ctx context.Context) (metricRulesetPayload, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	aggregationModels := metricRulesetListElements[metricRulesetAggregationRuleModel](ctx, model.AggregationRules, &diagnostics)
	exceptionModels := metricRulesetListElements[metricRulesetExceptionRuleModel](ctx, model.ExceptionRules, &diagnostics)
	routingModel := metricRulesetObjectValue[metricRulesetRoutingRuleModel](ctx, model.RoutingRule, &diagnostics)

	payload := metricRulesetPayload{
		MetricName:  model.MetricName.ValueString(),
		Description: metricRulesetStringPointer(model.Description),
		RoutingRule: metricruleset.RoutingRule{Destination: metricRulesetStringPointer(routingModel.Destination)},
	}
	for _, rule := range aggregationModels {
		payload.AggregationRules = append(payload.AggregationRules, rule.toAPI(ctx, &diagnostics))
	}
	for _, rule := range exceptionModels {
		payload.ExceptionRules = append(payload.ExceptionRules, rule.toAPI(ctx, &diagnostics))
	}
	return payload, diagnostics
}

func (model metricRulesetAggregationRuleModel) toAPI(ctx context.Context, diagnostics *diag.Diagnostics) metricruleset.AggregationRule {
	matcher := metricRulesetObjectValue[metricRulesetMatcherModel](ctx, model.Matcher, diagnostics).toAPI(ctx, diagnostics)
	aggregator := metricRulesetObjectValue[metricRulesetAggregatorModel](ctx, model.Aggregator, diagnostics).toAPI(ctx, diagnostics)
	return metricruleset.AggregationRule{
		Name:        metricRulesetStringPointer(model.Name),
		Description: metricRulesetStringPointer(model.Description),
		Enabled:     model.Enabled.ValueBool(),
		Matcher:     metricruleset.DimensionMatcherAsMetricMatcher(&matcher),
		Aggregator:  metricruleset.RollupAggregatorAsMetricAggregator(&aggregator),
	}
}

func (model metricRulesetExceptionRuleModel) toAPI(ctx context.Context, diagnostics *diag.Diagnostics) metricruleset.ExceptionRule {
	matcher := metricRulesetObjectValue[metricRulesetMatcherModel](ctx, model.Matcher, diagnostics).toAPI(ctx, diagnostics)
	return metricruleset.ExceptionRule{
		Name:        model.Name.ValueString(),
		Description: metricRulesetStringPointer(model.Description),
		Enabled:     model.Enabled.ValueBool(),
		Matcher:     matcher,
		Restoration: metricRulesetRestorationToAPI(ctx, model.Restoration, diagnostics),
	}
}

func (model metricRulesetMatcherModel) toAPI(ctx context.Context, diagnostics *diag.Diagnostics) metricruleset.DimensionMatcher {
	filterModels := metricRulesetListElements[metricRulesetFilterModel](ctx, model.Filters, diagnostics)
	filters := make([]metricruleset.PropertyFilter, 0, len(filterModels))
	for _, filter := range filterModels {
		values := metricRulesetSetStrings(ctx, filter.PropertyValue, diagnostics)
		filters = append(filters, metricruleset.PropertyFilter{
			Property:      metricRulesetStringPointer(filter.Property),
			PropertyValue: values,
			NOT:           metricRulesetBoolPointer(filter.Not),
		})
	}
	return metricruleset.DimensionMatcher{Type: model.Type.ValueString(), Filters: filters}
}

func (model metricRulesetAggregatorModel) toAPI(ctx context.Context, diagnostics *diag.Diagnostics) metricruleset.RollupAggregator {
	return metricruleset.RollupAggregator{
		Type:           model.Type.ValueString(),
		OutputName:     model.OutputName.ValueString(),
		Dimensions:     metricRulesetSetStrings(ctx, model.Dimensions, diagnostics),
		DropDimensions: metricRulesetBoolPointer(model.DropDimensions),
	}
}

func metricRulesetRestorationToAPI(ctx context.Context, value types.Object, diagnostics *diag.Diagnostics) *metricruleset.ExceptionRuleRestorationFields {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	model := metricRulesetObjectValue[metricRulesetRestorationModel](ctx, value, diagnostics)
	return &metricruleset.ExceptionRuleRestorationFields{
		RestorationId: metricRulesetStringPointer(model.RestorationID),
		StartTime:     metricRulesetInt64Pointer(model.StartTime),
		StopTime:      metricRulesetInt64Pointer(model.StopTime),
	}
}

func (model *resourceMetricRulesetModel) updateFromAPI(ctx context.Context, response metricRulesetResponse) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if value, ok := response.GetIdOk(); ok && *value != "" {
		model.ID = types.StringValue(*value)
	}
	if value, ok := response.GetMetricNameOk(); ok {
		model.MetricName = types.StringValue(*value)
	}
	if value, ok := response.GetDescriptionOk(); ok {
		model.Description = types.StringValue(*value)
	}
	if value, ok := response.GetVersionOk(); ok {
		model.Version = types.StringValue(strconv.FormatInt(*value, 10))
	}
	if rules, ok := response.GetAggregationRulesOk(); ok {
		model.AggregationRules = metricRulesetAggregationRulesFromAPI(ctx, rules, &diagnostics)
	}
	if rules, ok := response.GetExceptionRulesOk(); ok {
		model.ExceptionRules = metricRulesetExceptionRulesFromAPI(ctx, rules, &diagnostics)
	}
	if value, ok := response.GetRoutingRuleOk(); ok {
		model.RoutingRule = metricRulesetRoutingRuleFromAPI(ctx, value, &diagnostics)
	}
	creator, _ := response.GetCreatorOk()
	created, _ := response.GetCreatedOk()
	updater, _ := response.GetLastUpdatedByOk()
	updated, _ := response.GetLastUpdatedOk()
	updaterName, _ := response.GetLastUpdatedByNameOk()
	model.Creator = optionalStringValue(model.Creator, creator)
	model.Created = optionalInt64Value(model.Created, created)
	model.LastUpdatedBy = optionalStringValue(model.LastUpdatedBy, updater)
	model.LastUpdated = optionalInt64Value(model.LastUpdated, updated)
	model.LastUpdatedByName = optionalStringValue(model.LastUpdatedByName, updaterName)
	return diagnostics
}

func metricRulesetAggregationRulesFromAPI(ctx context.Context, rules []metricruleset.AggregationRule, diagnostics *diag.Diagnostics) types.List {
	values := make([]metricRulesetAggregationRuleModel, 0, len(rules))
	for i, rule := range rules {
		if rule.Matcher.DimensionMatcher == nil || rule.Aggregator.RollupAggregator == nil {
			diagnostics.AddError("Invalid metric aggregation rule response", fmt.Sprintf("Aggregation rule at index %d is missing its matcher or aggregator.", i))
			continue
		}
		values = append(values, metricRulesetAggregationRuleModel{
			Name:        metricRulesetStringValue(rule.Name),
			Description: metricRulesetStringValue(rule.Description),
			Enabled:     types.BoolValue(rule.Enabled),
			Matcher:     metricRulesetMatcherFromAPI(ctx, rule.Matcher.DimensionMatcher, diagnostics),
			Aggregator:  metricRulesetAggregatorFromAPI(ctx, rule.Aggregator.RollupAggregator, diagnostics),
		})
	}
	return metricRulesetListValue(ctx, metricRulesetAggregationRuleAttributeTypes, values, diagnostics)
}

func metricRulesetExceptionRulesFromAPI(ctx context.Context, rules []metricruleset.ExceptionRule, diagnostics *diag.Diagnostics) types.List {
	values := make([]metricRulesetExceptionRuleModel, 0, len(rules))
	for _, rule := range rules {
		values = append(values, metricRulesetExceptionRuleModel{
			Name:        types.StringValue(rule.Name),
			Description: metricRulesetStringValue(rule.Description),
			Enabled:     types.BoolValue(rule.Enabled),
			Matcher:     metricRulesetMatcherFromAPI(ctx, &rule.Matcher, diagnostics),
			Restoration: metricRulesetRestorationFromAPI(ctx, rule.Restoration, diagnostics),
		})
	}
	return metricRulesetListValue(ctx, metricRulesetExceptionRuleAttributeTypes, values, diagnostics)
}

func metricRulesetMatcherFromAPI(ctx context.Context, matcher *metricruleset.DimensionMatcher, diagnostics *diag.Diagnostics) types.Object {
	if matcher == nil || matcher.Type == "" {
		diagnostics.AddError("Invalid metric ruleset matcher response", "The metric ruleset API returned a matcher without a type.")
		return types.ObjectNull(metricRulesetMatcherAttributeTypes)
	}
	filters := make([]metricRulesetFilterModel, 0, len(matcher.Filters))
	for i, filter := range matcher.Filters {
		if filter.Property == nil || filter.NOT == nil {
			diagnostics.AddError("Invalid metric ruleset filter response", fmt.Sprintf("Filter at index %d is missing its property or not flag.", i))
			continue
		}
		values, valueDiagnostics := types.SetValueFrom(ctx, types.StringType, filter.PropertyValue)
		diagnostics.Append(valueDiagnostics...)
		filters = append(filters, metricRulesetFilterModel{
			Not:           types.BoolValue(*filter.NOT),
			Property:      types.StringValue(*filter.Property),
			PropertyValue: values,
		})
	}
	filterList := metricRulesetListValue(ctx, metricRulesetFilterAttributeTypes, filters, diagnostics)
	return metricRulesetObjectFrom(ctx, metricRulesetMatcherAttributeTypes, metricRulesetMatcherModel{
		Type: types.StringValue(matcher.Type), Filters: filterList,
	}, diagnostics)
}

func metricRulesetAggregatorFromAPI(ctx context.Context, aggregator *metricruleset.RollupAggregator, diagnostics *diag.Diagnostics) types.Object {
	if aggregator == nil || aggregator.DropDimensions == nil {
		diagnostics.AddError("Invalid metric ruleset aggregator response", "The metric ruleset API returned an aggregator without its drop_dimensions flag.")
		return types.ObjectNull(metricRulesetAggregatorAttributeTypes)
	}
	dimensions, dimensionDiagnostics := types.SetValueFrom(ctx, types.StringType, aggregator.Dimensions)
	diagnostics.Append(dimensionDiagnostics...)
	return metricRulesetObjectFrom(ctx, metricRulesetAggregatorAttributeTypes, metricRulesetAggregatorModel{
		Type:           types.StringValue(aggregator.Type),
		OutputName:     types.StringValue(aggregator.OutputName),
		Dimensions:     dimensions,
		DropDimensions: types.BoolValue(*aggregator.DropDimensions),
	}, diagnostics)
}

func metricRulesetRestorationFromAPI(ctx context.Context, restoration *metricruleset.ExceptionRuleRestorationFields, diagnostics *diag.Diagnostics) types.Object {
	if restoration == nil || (restoration.RestorationId == nil && restoration.StartTime == nil && restoration.StopTime == nil) {
		return types.ObjectNull(metricRulesetRestorationAttributeTypes)
	}
	if restoration.StartTime == nil {
		diagnostics.AddError("Invalid metric ruleset restoration response", "The metric ruleset API returned restoration metadata without a start time.")
		return types.ObjectNull(metricRulesetRestorationAttributeTypes)
	}
	return metricRulesetObjectFrom(ctx, metricRulesetRestorationAttributeTypes, metricRulesetRestorationModel{
		RestorationID: optionalStringValue(types.StringNull(), restoration.RestorationId),
		StartTime:     types.Int64Value(*restoration.StartTime),
		StopTime:      optionalInt64Value(types.Int64Null(), restoration.StopTime),
	}, diagnostics)
}

func metricRulesetRoutingRuleFromAPI(ctx context.Context, routing *metricruleset.RoutingRule, diagnostics *diag.Diagnostics) types.Object {
	if routing == nil || routing.Destination == nil {
		diagnostics.AddError("Invalid metric ruleset routing response", "The metric ruleset API returned no routing destination.")
		return types.ObjectNull(metricRulesetRoutingRuleAttributeTypes)
	}
	return metricRulesetObjectFrom(ctx, metricRulesetRoutingRuleAttributeTypes, metricRulesetRoutingRuleModel{
		Destination: types.StringValue(*routing.Destination),
	}, diagnostics)
}

func metricRulesetListElements[T any](ctx context.Context, value types.List, diagnostics *diag.Diagnostics) []T {
	if value.IsNull() {
		return []T{}
	}
	var values []T
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	return values
}

func metricRulesetObjectValue[T any](ctx context.Context, value types.Object, diagnostics *diag.Diagnostics) T {
	var model T
	diagnostics.Append(value.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	return model
}

func metricRulesetSetStrings(ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []string {
	var values []string
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	return values
}

func metricRulesetObjectFrom[T any](ctx context.Context, attributeTypes map[string]attr.Type, model T, diagnostics *diag.Diagnostics) types.Object {
	value, valueDiagnostics := types.ObjectValueFrom(ctx, attributeTypes, model)
	diagnostics.Append(valueDiagnostics...)
	return value
}

func metricRulesetListValue[T any](ctx context.Context, attributeTypes map[string]attr.Type, values []T, diagnostics *diag.Diagnostics) types.List {
	value, valueDiagnostics := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, values)
	diagnostics.Append(valueDiagnostics...)
	return value
}

func metricRulesetStringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func metricRulesetBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func metricRulesetInt64Pointer(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueInt64()
	return &result
}

func metricRulesetStringValue(value *string) types.String {
	if value == nil {
		return types.StringValue("")
	}
	return types.StringValue(*value)
}

var metricRulesetFilterAttributeTypes = map[string]attr.Type{
	"not": types.BoolType, "property": types.StringType, "property_value": types.SetType{ElemType: types.StringType},
}

var metricRulesetMatcherAttributeTypes = map[string]attr.Type{
	"type": types.StringType, "filters": types.ListType{ElemType: types.ObjectType{AttrTypes: metricRulesetFilterAttributeTypes}},
}

var metricRulesetAggregatorAttributeTypes = map[string]attr.Type{
	"type": types.StringType, "output_name": types.StringType,
	"dimensions": types.SetType{ElemType: types.StringType}, "drop_dimensions": types.BoolType,
}

var metricRulesetRestorationAttributeTypes = map[string]attr.Type{
	"restoration_id": types.StringType, "start_time": types.Int64Type, "stop_time": types.Int64Type,
}

var metricRulesetRoutingRuleAttributeTypes = map[string]attr.Type{"destination": types.StringType}

var metricRulesetAggregationRuleAttributeTypes = map[string]attr.Type{
	"name": types.StringType, "description": types.StringType, "enabled": types.BoolType,
	"matcher":    types.ObjectType{AttrTypes: metricRulesetMatcherAttributeTypes},
	"aggregator": types.ObjectType{AttrTypes: metricRulesetAggregatorAttributeTypes},
}

var metricRulesetExceptionRuleAttributeTypes = map[string]attr.Type{
	"name": types.StringType, "description": types.StringType, "enabled": types.BoolType,
	"matcher":     types.ObjectType{AttrTypes: metricRulesetMatcherAttributeTypes},
	"restoration": types.ObjectType{AttrTypes: metricRulesetRestorationAttributeTypes},
}
