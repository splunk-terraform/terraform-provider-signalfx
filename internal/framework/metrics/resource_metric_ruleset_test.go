// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	metricruleset "github.com/signalfx/signalfx-go/metric_ruleset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceMetricRulesetMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceMetricRuleset()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_metric_ruleset", metadata.TypeName)

	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	assert.NotEmpty(t, schemaResponse.Schema.Description)
	assert.IsType(t, schema.Int64Attribute{}, schemaResponse.Schema.Attributes["created"])
	aggregation := schemaResponse.Schema.Blocks["aggregation_rules"].(schema.ListNestedBlock)
	assert.IsType(t, schema.SingleNestedBlock{}, aggregation.NestedObject.Blocks["matcher"])
	assert.IsType(t, schema.SingleNestedBlock{}, aggregation.NestedObject.Blocks["aggregator"])
	exception := schemaResponse.Schema.Blocks["exception_rules"].(schema.ListNestedBlock)
	assert.IsType(t, schema.SingleNestedBlock{}, exception.NestedObject.Blocks["restoration"])
	routing := schemaResponse.Schema.Blocks["routing_rule"].(schema.SingleNestedBlock)
	assert.Len(t, routing.Validators, 1)
}

func TestResourceMetricRulesetModelToAPI(t *testing.T) {
	t.Parallel()
	model := metricRulesetTestModel(t)
	create, diagnostics := model.createRequest(context.Background())
	require.False(t, diagnostics.HasError())
	assert.Equal(t, "demo.metric", create.MetricName)
	assert.Equal(t, "Archived", *create.RoutingRule.Destination)
	require.Len(t, create.AggregationRules, 1)
	assert.Equal(t, "dimension", create.AggregationRules[0].Matcher.DimensionMatcher.Type)
	assert.Equal(t, "demo.metric.by.service", create.AggregationRules[0].Aggregator.RollupAggregator.OutputName)
	require.Len(t, create.ExceptionRules, 1)
	assert.Equal(t, int64(100), *create.ExceptionRules[0].Restoration.StartTime)
	assert.Nil(t, create.ExceptionRules[0].Restoration.StopTime)

	version := int64(7)
	update, diagnostics := model.updateRequest(context.Background(), &version)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, int64(7), *update.Version)
	assert.Equal(t, "demo.metric", *update.MetricName)
}

func TestResourceMetricRulesetModelFromAPI(t *testing.T) {
	t.Parallel()
	model := metricRulesetTestModel(t)
	id, creator, updater, updaterName := "ruleset-id", "creator", "updater", "Updater Name"
	description, metricName, destination := "from API", "api.metric", "RealTime"
	created, updated, version := int64(10), int64(20), int64(3)
	name, property, restorationID := "aggregate", "realm", "restore-id"
	not, drop := false, false
	start := int64(100)
	response := &metricruleset.CreateMetricRulesetResponse{
		Id: &id, MetricName: &metricName, Description: &description, Version: &version,
		Creator: &creator, Created: &created, LastUpdatedBy: &updater, LastUpdated: &updated, LastUpdatedByName: &updaterName,
		AggregationRules: []metricruleset.AggregationRule{{
			Name: &name, Enabled: true,
			Matcher:    metricruleset.DimensionMatcherAsMetricMatcher(&metricruleset.DimensionMatcher{Type: "dimension", Filters: []metricruleset.PropertyFilter{{Property: &property, PropertyValue: []string{"us0"}, NOT: &not}}}),
			Aggregator: metricruleset.RollupAggregatorAsMetricAggregator(&metricruleset.RollupAggregator{Type: "rollup", OutputName: "api.output", Dimensions: []string{"service"}, DropDimensions: &drop}),
		}},
		ExceptionRules: []metricruleset.ExceptionRule{{
			Name: "exception", Enabled: true,
			Matcher:     metricruleset.DimensionMatcher{Type: "dimension"},
			Restoration: &metricruleset.ExceptionRuleRestorationFields{RestorationId: &restorationID, StartTime: &start},
		}},
		RoutingRule: &metricruleset.RoutingRule{Destination: &destination},
	}
	diagnostics := model.updateFromAPI(context.Background(), response)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.StringValue("ruleset-id"), model.ID)
	assert.Equal(t, types.StringValue("3"), model.Version)
	assert.Equal(t, types.Int64Value(10), model.Created)
	assert.Equal(t, types.StringValue("Updater Name"), model.LastUpdatedByName)

	var exceptions []metricRulesetExceptionRuleModel
	require.False(t, model.ExceptionRules.ElementsAs(context.Background(), &exceptions, false).HasError())
	require.Len(t, exceptions, 1)
	var restoration metricRulesetRestorationModel
	require.False(t, exceptions[0].Restoration.As(context.Background(), &restoration, structObjectAsOptions()).HasError())
	assert.Equal(t, types.StringValue("restore-id"), restoration.RestorationID)
	assert.True(t, restoration.StopTime.IsNull())
}

func TestResourceMetricRulesetModelMalformedAPI(t *testing.T) {
	t.Parallel()
	model := metricRulesetTestModel(t)
	id, version, metricName := "ruleset-id", int64(1), "demo.metric"
	property := "realm"
	response := &metricruleset.CreateMetricRulesetResponse{
		Id: &id, Version: &version, MetricName: &metricName,
		AggregationRules: []metricruleset.AggregationRule{{}},
		ExceptionRules: []metricruleset.ExceptionRule{
			{Name: "bad-matcher", Matcher: metricruleset.DimensionMatcher{}},
			{Name: "bad-filter", Matcher: metricruleset.DimensionMatcher{Type: "dimension", Filters: []metricruleset.PropertyFilter{{Property: &property}}}},
			{Name: "bad-restoration", Matcher: metricruleset.DimensionMatcher{Type: "dimension"}, Restoration: &metricruleset.ExceptionRuleRestorationFields{RestorationId: &id}},
		},
		RoutingRule: &metricruleset.RoutingRule{},
	}
	diagnostics := model.updateFromAPI(context.Background(), response)
	assert.True(t, diagnostics.HasError())

	drop := false
	assert.True(t, metricRulesetAggregatorFromAPI(context.Background(), &metricruleset.RollupAggregator{}, &diagnostics).IsNull())
	assert.False(t, metricRulesetAggregatorFromAPI(context.Background(), &metricruleset.RollupAggregator{DropDimensions: &drop}, &diagnostics).IsNull())
	assert.True(t, metricRulesetMatcherFromAPI(context.Background(), nil, &diagnostics).IsNull())
	assert.True(t, metricRulesetRoutingRuleFromAPI(context.Background(), nil, &diagnostics).IsNull())
	assert.True(t, metricRulesetRestorationFromAPI(context.Background(), &metricruleset.ExceptionRuleRestorationFields{}, &diagnostics).IsNull())
}

func TestResourceMetricRulesetRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceMetricRuleset{}
	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	plan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}
	state := tfsdk.State{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}
	createResponse := &resource.CreateResponse{}
	implementation.Create(ctx, resource.CreateRequest{Plan: plan}, createResponse)
	assert.True(t, createResponse.Diagnostics.HasError())
	readResponse := &resource.ReadResponse{}
	implementation.Read(ctx, resource.ReadRequest{State: state}, readResponse)
	assert.True(t, readResponse.Diagnostics.HasError())
	updateResponse := &resource.UpdateResponse{}
	implementation.Update(ctx, resource.UpdateRequest{Plan: plan}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: state}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourceMetricRulesetMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := metricruleset.MetricRuleset{}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		assert.NoError(t, json.NewEncoder(w).Encode(current))
	}
	endpoints := map[string]http.Handler{
		"POST /v2/metricruleset": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload metricruleset.CreateMetricRulesetRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, "demo.metric", payload.MetricName)
			assert.Equal(t, "Archived", *payload.RoutingRule.Destination)
			assert.Equal(t, int64(100), *payload.ExceptionRules[0].Restoration.StartTime)
			assert.Nil(t, payload.ExceptionRules[0].Restoration.StopTime)
			id, creator, updater, updaterName := "ruleset-id", "creator", "creator", "Creator Name"
			created, updated, version := int64(10), int64(10), int64(1)
			restorationID := "restore-id"
			payload.ExceptionRules[0].Restoration.RestorationId = &restorationID
			mu.Lock()
			current = metricruleset.MetricRuleset{
				Id: &id, MetricName: &payload.MetricName, Description: payload.Description, Version: &version,
				AggregationRules: payload.AggregationRules, ExceptionRules: payload.ExceptionRules, RoutingRule: &payload.RoutingRule,
				Creator: &creator, Created: &created, LastUpdatedBy: &updater, LastUpdated: &updated, LastUpdatedByName: &updaterName,
			}
			mu.Unlock()
			writeCurrent(w)
		}),
		"GET /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload metricruleset.UpdateMetricRulesetRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, int64(1), *payload.Version)
			assert.Equal(t, "Drop", *payload.RoutingRule.Destination)
			assert.Equal(t, int64(200), *payload.ExceptionRules[0].Restoration.StopTime)
			id, updater, updaterName := "ruleset-id", "updater", "Updater Name"
			updated, version := int64(20), int64(2)
			mu.Lock()
			current.Id, current.Version = &id, &version
			current.MetricName, current.Description = payload.MetricName, payload.Description
			current.AggregationRules, current.ExceptionRules, current.RoutingRule = payload.AggregationRules, payload.ExceptionRules, payload.RoutingRule
			current.LastUpdatedBy, current.LastUpdated, current.LastUpdatedByName = &updater, &updated, &updaterName
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceMetricRuleset)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/metric_ruleset_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "id", "ruleset-id"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "version", "1"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "aggregation_rules.0.matcher.type", "dimension"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "exception_rules.0.restoration.restoration_id", "restore-id"),
			)},
			{ConfigFile: config.StaticFile("testdata/metric_ruleset_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "version", "2"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "last_updated_by_name", "Updater Name"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "routing_rule.destination", "Drop"),
				testresource.TestCheckResourceAttr("signalfx_metric_ruleset.test", "exception_rules.0.restoration.stop_time", "200"),
			)},
			{ConfigFile: config.StaticFile("testdata/metric_ruleset_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_metric_ruleset.test", ImportState: true, ImportStateId: "ruleset-id", ImportStateVerify: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceMetricRulesetRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	id, metricName, destination, version := "ruleset-id", "demo.metric", "Archived", int64(1)
	current := metricruleset.MetricRuleset{Id: &id, MetricName: &metricName, Version: &version, RoutingRule: &metricruleset.RoutingRule{Destination: &destination}}
	endpoints := map[string]http.Handler{
		"POST /v2/metricruleset": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(current)) }),
		"GET /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				assert.NoError(t, json.NewEncoder(w).Encode(current))
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceMetricRuleset)),
		Steps: []testresource.TestStep{
			{Config: minimalMetricRulesetConfig},
			{Config: minimalMetricRulesetConfig, PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceMetricRulesetErrors(t *testing.T) {
	id, metricName, destination, version := "ruleset-id", "demo.metric", "Archived", int64(1)
	valid := metricruleset.MetricRuleset{Id: &id, MetricName: &metricName, Version: &version, RoutingRule: &metricruleset.RoutingRule{Destination: &destination}}
	for _, test := range []struct {
		name, config string
		endpoints    map[string]http.Handler
		error        *regexp.Regexp
	}{
		{name: "create API error", config: minimalMetricRulesetConfig, endpoints: map[string]http.Handler{
			"POST /v2/metricruleset": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", http.StatusBadGateway) }),
		}, error: regexp.MustCompile(`status code 502`)},
		{name: "nil create response", config: minimalMetricRulesetConfig, endpoints: map[string]http.Handler{
			"POST /v2/metricruleset": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`null`)) }),
		}, error: regexp.MustCompile(`no resource identifier`)},
		{name: "missing create ID", config: minimalMetricRulesetConfig, endpoints: map[string]http.Handler{
			"POST /v2/metricruleset": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) }),
		}, error: regexp.MustCompile(`no resource identifier`)},
		{name: "missing routing", config: `resource "signalfx_metric_ruleset" "test" { metric_name = "demo.metric" }`, error: regexp.MustCompile(`routing_rule`)},
		{name: "invalid destination", config: `resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  routing_rule { destination = "invalid" }
}`, error: regexp.MustCompile(`RealTime`)},
		{name: "invalid restoration timestamp", config: `resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  exception_rules {
    enabled = true
    matcher { type = "dimension" }
    restoration { start_time = "not-a-number" }
  }
  routing_rule { destination = "Archived" }
}`, error: regexp.MustCompile(`number is required`)},
		{name: "missing matcher", config: `resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  aggregation_rules {
    enabled = true
    aggregator {
      type = "rollup"
      output_name = "out"
      dimensions = ["host"]
      drop_dimensions = false
    }
  }
  routing_rule { destination = "Archived" }
}`, error: regexp.MustCompile(`matcher`)},
		{name: "read API error", config: minimalMetricRulesetConfig, endpoints: map[string]http.Handler{
			"POST /v2/metricruleset":              http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(valid) }),
			"GET /v2/metricruleset/ruleset-id":    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", http.StatusBadGateway) }),
			"DELETE /v2/metricruleset/ruleset-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		}, error: regexp.MustCompile(`status code 502`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockResources(NewResourceMetricRuleset)),
				Steps:                    []testresource.TestStep{{Config: test.config, ExpectError: test.error}},
			})
		})
	}
}

func metricRulesetTestModel(t *testing.T) resourceMetricRulesetModel {
	t.Helper()
	ctx := context.Background()
	var diagnostics diag.Diagnostics
	filter := metricRulesetFilterModel{
		Not: types.BoolValue(false), Property: types.StringValue("realm"),
		PropertyValue: metricRulesetTestSet(t, "us0"),
	}
	matcher := metricRulesetObjectFrom(ctx, metricRulesetMatcherAttributeTypes, metricRulesetMatcherModel{
		Type: types.StringValue("dimension"), Filters: metricRulesetListValue(ctx, metricRulesetFilterAttributeTypes, []metricRulesetFilterModel{filter}, &diagnostics),
	}, &diagnostics)
	aggregator := metricRulesetObjectFrom(ctx, metricRulesetAggregatorAttributeTypes, metricRulesetAggregatorModel{
		Type: types.StringValue("rollup"), OutputName: types.StringValue("demo.metric.by.service"),
		Dimensions: metricRulesetTestSet(t, "service"), DropDimensions: types.BoolValue(false),
	}, &diagnostics)
	restoration := metricRulesetObjectFrom(ctx, metricRulesetRestorationAttributeTypes, metricRulesetRestorationModel{
		RestorationID: types.StringUnknown(), StartTime: types.Int64Value(100), StopTime: types.Int64Null(),
	}, &diagnostics)
	aggregations := metricRulesetListValue(ctx, metricRulesetAggregationRuleAttributeTypes, []metricRulesetAggregationRuleModel{{
		Name: types.StringValue("aggregate"), Description: types.StringValue("description"), Enabled: types.BoolValue(true), Matcher: matcher, Aggregator: aggregator,
	}}, &diagnostics)
	exceptions := metricRulesetListValue(ctx, metricRulesetExceptionRuleAttributeTypes, []metricRulesetExceptionRuleModel{{
		Name: types.StringValue("exception"), Description: types.StringValue("description"), Enabled: types.BoolValue(true), Matcher: matcher, Restoration: restoration,
	}}, &diagnostics)
	routing := metricRulesetObjectFrom(ctx, metricRulesetRoutingRuleAttributeTypes, metricRulesetRoutingRuleModel{Destination: types.StringValue("Archived")}, &diagnostics)
	require.False(t, diagnostics.HasError())
	return resourceMetricRulesetModel{
		ID: types.StringValue("ruleset-id"), MetricName: types.StringValue("demo.metric"), Description: types.StringValue("description"),
		AggregationRules: aggregations, ExceptionRules: exceptions, RoutingRule: routing,
	}
}

func metricRulesetTestSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	value, diagnostics := types.SetValueFrom(context.Background(), types.StringType, values)
	require.False(t, diagnostics.HasError())
	return value
}

func structObjectAsOptions() basetypes.ObjectAsOptions {
	return basetypes.ObjectAsOptions{}
}

const minimalMetricRulesetConfig = `resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  routing_rule { destination = "Archived" }
}`
