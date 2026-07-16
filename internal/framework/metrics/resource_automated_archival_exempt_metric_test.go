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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	automatedarchival "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceAutomatedArchivalExemptMetricMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceAutomatedArchivalExemptMetric()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_automated_archival_exempt_metric", metadata.TypeName)

	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	assert.NotEmpty(t, schemaResponse.Schema.Description)
	assert.IsType(t, schema.StringAttribute{}, schemaResponse.Schema.Attributes["id"])
	block, ok := schemaResponse.Schema.Blocks["exempt_metrics"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Len(t, block.Validators, 1)
	assert.Len(t, block.PlanModifiers, 1)
	assert.IsType(t, schema.StringAttribute{}, block.NestedObject.Attributes["name"])
	assert.IsType(t, schema.Int64Attribute{}, block.NestedObject.Attributes["created"])
}

func TestResourceAutomatedArchivalExemptMetricModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceAutomatedArchivalExemptMetricModel{
		ID: types.StringValue("id-a,id-b"),
		ExemptMetrics: exemptMetricListValue(t,
			automatedArchivalExemptMetricModel{Name: types.StringValue("metric.cpu")},
			automatedArchivalExemptMetricModel{Name: types.StringValue("metric.memory")},
		),
	}
	payload, diagnostics := model.toAPI(ctx)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, []automatedarchival.ExemptMetric{{Name: "metric.cpu"}, {Name: "metric.memory"}}, payload)

	creator, updater := "creator", "updater"
	created, updated := int64(100), int64(200)
	details := []automatedarchival.ExemptMetric{{
		Name: "metric.cpu", Creator: &creator, LastUpdatedBy: &updater, Created: &created, LastUpdated: &updated,
	}}
	require.False(t, model.updateFromAPI(ctx, details).HasError())
	var values []automatedArchivalExemptMetricModel
	require.False(t, model.ExemptMetrics.ElementsAs(ctx, &values, false).HasError())
	require.Len(t, values, 1)
	assert.Equal(t, types.StringValue("creator"), values[0].Creator)
	assert.Equal(t, types.Int64Value(200), values[0].LastUpdated)

	unknownModel := resourceAutomatedArchivalExemptMetricModel{ExemptMetrics: types.ListUnknown(types.ObjectType{AttrTypes: automatedArchivalExemptMetricAttributeTypes})}
	_, diagnostics = unknownModel.toAPI(ctx)
	assert.True(t, diagnostics.HasError())
	nullModel := resourceAutomatedArchivalExemptMetricModel{ExemptMetrics: types.ListNull(types.ObjectType{AttrTypes: automatedArchivalExemptMetricAttributeTypes})}
	_, diagnostics = nullModel.toAPI(ctx)
	assert.True(t, diagnostics.HasError())
	unknownName := resourceAutomatedArchivalExemptMetricModel{ExemptMetrics: exemptMetricListValue(t,
		automatedArchivalExemptMetricModel{Name: types.StringUnknown()},
	)}
	_, diagnostics = unknownName.toAPI(ctx)
	assert.True(t, diagnostics.HasError())
}

func TestAutomatedArchivalExemptMetricIdentifiers(t *testing.T) {
	t.Parallel()
	ids, err := automatedArchivalExemptMetricIDs(" id-a,id-b ")
	require.NoError(t, err)
	assert.Equal(t, []string{"id-a", "id-b"}, ids)
	for _, invalid := range []string{"", " ", "id-a,", ",id-b", "id-a,,id-b"} {
		_, parseErr := automatedArchivalExemptMetricIDs(invalid)
		assert.Error(t, parseErr)
	}

	idA, idB, unrelated := "id-a", "id-b", "other"
	details := []automatedarchival.ExemptMetric{
		{Name: "unrelated", Id: &unrelated},
		{Name: "metric.memory", Id: &idB},
		{Name: "missing-id"},
		{Name: "metric.cpu", Id: &idA},
	}
	managed, found := filterAutomatedArchivalExemptMetrics(details, []string{"id-a", "missing", "id-b"})
	assert.Equal(t, []string{"id-a", "id-b"}, found)
	assert.Equal(t, []string{"metric.cpu", "metric.memory"}, []string{managed[0].Name, managed[1].Name})
	responseIDs, err := automatedArchivalExemptMetricResponseIDs(managed)
	require.NoError(t, err)
	assert.Equal(t, found, responseIDs)
	_, err = automatedArchivalExemptMetricResponseIDs([]automatedarchival.ExemptMetric{{Name: "no-id"}})
	assert.Error(t, err)
}

func TestResourceAutomatedArchivalExemptMetricRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceAutomatedArchivalExemptMetric{}
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
	implementation.Update(ctx, resource.UpdateRequest{State: state}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: state}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourceAutomatedArchivalExemptMetricMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := []automatedarchival.ExemptMetric{}
	deleted := false
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload []automatedarchival.ExemptMetric
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, []string{"metric.cpu", "metric.memory"}, []string{payload[0].Name, payload[1].Name})
			assert.Nil(t, payload[0].Creator)
			idA, idB, creator, updater := "id-a", "id-b", "creator", "updater"
			created, updated := int64(100), int64(200)
			mu.Lock()
			current = []automatedarchival.ExemptMetric{
				{Name: "metric.cpu", Id: &idA, Creator: &creator, Created: &created},
				{Name: "metric.memory", Id: &idB, LastUpdatedBy: &updater, LastUpdated: &updated},
			}
			response := append([]automatedarchival.ExemptMetric(nil), current...)
			mu.Unlock()
			assert.NoError(t, json.NewEncoder(w).Encode(response))
		}),
		"GET /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			unrelatedID := "unrelated"
			mu.Lock()
			response := []automatedarchival.ExemptMetric{{Name: "unrelated.metric", Id: &unrelatedID}, current[1], current[0]}
			mu.Unlock()
			assert.NoError(t, json.NewEncoder(w).Encode(response))
		}),
		"DELETE /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload automatedarchival.ExemptMetricDeleteRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, []string{"id-a", "id-b"}, payload.Ids)
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalExemptMetric)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_automated_archival_exempt_metric.test", "id", "id-a,id-b"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_exempt_metric.test", "exempt_metrics.0.name", "metric.cpu"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_exempt_metric.test", "exempt_metrics.0.creator", "creator"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_exempt_metric.test", "exempt_metrics.1.last_updated", "200"),
			)},
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf"), PlanOnly: true},
			{ResourceName: "signalfx_automated_archival_exempt_metric.test", ImportState: true, ImportStateId: "id-a,id-b", ImportStateVerify: true},
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_replace.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceAutomatedArchivalExemptMetricPartialRemoteDeletion(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	idA, idB := "id-a", "id-b"
	full := []automatedarchival.ExemptMetric{{Name: "metric.cpu", Id: &idA}, {Name: "metric.memory", Id: &idB}}
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(full)) }),
		"GET /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			response := full
			if getCalls > 1 {
				response = full[1:]
			}
			assert.NoError(t, json.NewEncoder(w).Encode(response))
		}),
		"DELETE /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload automatedarchival.ExemptMetricDeleteRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.NotEmpty(t, payload.Ids)
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalExemptMetric)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf")},
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, getCalls, 1)
}

func TestResourceAutomatedArchivalExemptMetricCompleteRemoteDeletion(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	idA, idB := "id-a", "id-b"
	created := []automatedarchival.ExemptMetric{{Name: "metric.cpu", Id: &idA}, {Name: "metric.memory", Id: &idB}}
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(created)) }),
		"GET /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				assert.NoError(t, json.NewEncoder(w).Encode(created))
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode([]automatedarchival.ExemptMetric{}))
		}),
		"DELETE /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalExemptMetric)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf")},
			{ConfigFile: config.StaticFile("testdata/automated_archival_exempt_metric_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceAutomatedArchivalExemptMetricErrors(t *testing.T) {
	valid := `resource "signalfx_automated_archival_exempt_metric" "test" {
  exempt_metrics { name = "metric.cpu" }
}`
	id := "id-a"
	for _, test := range []struct {
		name      string
		config    string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{name: "API error", config: valid, endpoints: map[string]http.Handler{
			"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", http.StatusBadGateway) }),
		}, error: regexp.MustCompile(`status code 502`)},
		{name: "nil API response", config: valid, endpoints: map[string]http.Handler{
			"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`null`)) }),
		}, error: regexp.MustCompile(`returned no exempt metrics`)},
		{name: "empty API response", config: valid, endpoints: map[string]http.Handler{
			"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) }),
		}, error: regexp.MustCompile(`returned no exempt metrics`)},
		{name: "missing response ID", config: valid, endpoints: map[string]http.Handler{
			"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode([]automatedarchival.ExemptMetric{{Name: "metric.cpu"}})
			}),
		}, error: regexp.MustCompile(`returned no ID`)},
		{name: "missing block", config: `resource "signalfx_automated_archival_exempt_metric" "test" {}`, error: regexp.MustCompile(`at least one known metric`)},
		{name: "read API error", config: valid, endpoints: map[string]http.Handler{
			"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode([]automatedarchival.ExemptMetric{{Name: "metric.cpu", Id: &id}})
			}),
			"GET /v2/automated-archival/exempt-metrics":    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", http.StatusBadGateway) }),
			"DELETE /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		}, error: regexp.MustCompile(`status code 502`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalExemptMetric)),
				Steps:                    []testresource.TestStep{{Config: test.config, ExpectError: test.error}},
			})
		})
	}
}

func TestResourceAutomatedArchivalExemptMetricDeleteError(t *testing.T) {
	var mu sync.Mutex
	deleteCalls := 0
	id := "id-a"
	created := []automatedarchival.ExemptMetric{{Name: "metric.cpu", Id: &id}}
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(created))
		}),
		"GET /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(created))
		}),
		"DELETE /v2/automated-archival/exempt-metrics": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			deleteCalls++
			if deleteCalls == 1 {
				http.Error(w, "failed", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalExemptMetric)),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_automated_archival_exempt_metric" "test" {
  exempt_metrics { name = "metric.cpu" }
}`},
			{Config: `resource "signalfx_automated_archival_exempt_metric" "test" {
  exempt_metrics { name = "metric.cpu" }
}`, Destroy: true, ExpectError: regexp.MustCompile(`status code 502`)},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, deleteCalls)
}

func exemptMetricListValue(t *testing.T, values ...automatedArchivalExemptMetricModel) types.List {
	t.Helper()
	value, diagnostics := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: automatedArchivalExemptMetricAttributeTypes}, values)
	require.False(t, diagnostics.HasError())
	return value
}
