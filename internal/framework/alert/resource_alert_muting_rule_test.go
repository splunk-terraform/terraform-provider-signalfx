// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/alertmuting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

var (
	filterObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"property": types.StringType, "property_value": types.StringType, "negated": types.BoolType,
	}}
	recurrenceObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"unit": types.StringType, "value": types.Int64Type,
	}}
)

func TestResourceAlertMutingRuleMetadata(t *testing.T) {
	t.Parallel()

	resp := &resource.MetadataResponse{}
	NewResourceAlertMutingRule().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_alert_muting_rule", resp.TypeName)
}

func TestResourceAlertMutingRuleSchema(t *testing.T) {
	t.Parallel()

	resp := &resource.SchemaResponse{}
	NewResourceAlertMutingRule().Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())
	require.Len(t, resp.Schema.Attributes, 6)
	require.Len(t, resp.Schema.Blocks, 2)
	assert.True(t, resp.Schema.Attributes["description"].IsRequired())
	assert.True(t, resp.Schema.Attributes["detectors"].IsOptional())
	assert.Contains(t, resp.Schema.Blocks, "filter")
	assert.Contains(t, resp.Schema.Blocks, "recurrence")
	assert.True(t, resp.Schema.Attributes["start_time"].IsRequired())
	assert.True(t, resp.Schema.Attributes["stop_time"].IsOptional())
	assert.True(t, resp.Schema.Attributes["effective_start_time"].IsComputed())
}

func TestAlertMutingRuleToRequest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	filters, diags := types.SetValueFrom(ctx, filterObjectType, []alertMutingRuleFilterModel{{
		Property: types.StringValue("host"), PropertyValue: types.StringValue("web-1"), Negated: types.BoolValue(true),
	}})
	require.False(t, diags.HasError())
	recurrence, diags := types.SetValueFrom(ctx, recurrenceObjectType, []alertMutingRuleRecurrenceModel{{
		Unit: types.StringValue("d"), Value: types.Int64Value(2),
	}})
	require.False(t, diags.HasError())
	detectors, diags := types.ListValueFrom(ctx, types.StringType, []string{"detector-1", "detector-2"})
	require.False(t, diags.HasError())

	model := alertMutingRuleModel{
		Description: types.StringValue("maintenance"), Detectors: detectors, Filter: filters,
		Recurrence: recurrence, StartTime: types.Int64Value(100), StopTime: types.Int64Value(200),
		EffectiveStartTime: types.Int64Value(123456),
	}

	payload, gotDiags := model.toRequest(ctx, false, time.Unix(50, 0))
	require.False(t, gotDiags.HasError())
	assert.Equal(t, int64(100000), payload.StartTime)
	assert.Equal(t, int64(200000), payload.StopTime)
	assert.Equal(t, &alertmuting.AlertMutingRuleRecurrence{Unit: "d", Value: 2}, payload.Recurrence)
	require.Len(t, payload.Filters, 2)
	assert.Equal(t, "host", payload.Filters[0].Property)
	assert.Equal(t, []string{"web-1"}, payload.Filters[0].PropertyValue.Values)
	assert.True(t, payload.Filters[0].NOT)
	assert.Equal(t, alertMutingDetectorIDProperty, payload.Filters[1].Property)
	assert.Equal(t, []string{"detector-1", "detector-2"}, payload.Filters[1].PropertyValue.Values)

	payload, gotDiags = model.toRequest(ctx, true, time.Unix(101, 0))
	require.False(t, gotDiags.HasError())
	assert.Equal(t, int64(123456), payload.StartTime, "past updates use the API effective start time")

	payload, gotDiags = model.toRequest(ctx, true, time.Unix(99, 0))
	require.False(t, gotDiags.HasError())
	assert.Equal(t, int64(100000), payload.StartTime, "future updates use the configured start time")
}

func TestAlertMutingRuleUpdateFromRule(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := alertMutingRuleModel{StartTime: types.Int64Value(100)}

	diags := model.updateFromRule(ctx, &alertmuting.AlertMutingRule{
		Id: "rule-1", Description: "maintenance", StartTime: 123456, StopTime: 200000,
		Filters: []*alertmuting.AlertMutingRuleFilter{
			{Property: alertMutingDetectorIDProperty, PropertyValue: alertmuting.StringOrArray{Values: []string{"detector-1", "detector-2"}}},
			{Property: "host", PropertyValue: alertmuting.StringOrArray{Values: []string{"web-1"}}, NOT: true},
		},
		Recurrence: &alertmuting.AlertMutingRuleRecurrence{Unit: "w", Value: 3},
	})
	require.False(t, diags.HasError())
	assert.Equal(t, "rule-1", model.ID.ValueString())
	assert.Equal(t, int64(100), model.StartTime.ValueInt64(), "configured start time must remain stable")
	assert.Equal(t, int64(123456), model.EffectiveStartTime.ValueInt64())
	assert.Equal(t, int64(200), model.StopTime.ValueInt64())

	var detectors []string
	diags = model.Detectors.ElementsAs(ctx, &detectors, false)
	require.False(t, diags.HasError())
	assert.Equal(t, []string{"detector-1", "detector-2"}, detectors)
	var filters []alertMutingRuleFilterModel
	diags = model.Filter.ElementsAs(ctx, &filters, false)
	require.False(t, diags.HasError())
	require.Len(t, filters, 1)
	assert.Equal(t, "host", filters[0].Property.ValueString())
	assert.True(t, filters[0].Negated.ValueBool())
}

func TestAlertMutingRuleRejectsArrayFilterValues(t *testing.T) {
	t.Parallel()
	model := alertMutingRuleModel{StartTime: types.Int64Value(1)}

	diags := model.updateFromRule(context.Background(), &alertmuting.AlertMutingRule{
		Id: "rule-1",
		Filters: []*alertmuting.AlertMutingRuleFilter{{
			Property: "host", PropertyValue: alertmuting.StringOrArray{Values: []string{"one", "two"}},
		}},
	})

	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "does not support arrays")
}

func TestAlertMutingRuleRejectsMalformedResponses(t *testing.T) {
	t.Parallel()
	model := alertMutingRuleModel{}

	diags := model.updateFromRule(context.Background(), nil)
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "no resource data")

	diags = model.updateFromRule(context.Background(), &alertmuting.AlertMutingRule{})
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "no resource identifier")

	diags = model.updateFromRule(context.Background(), &alertmuting.AlertMutingRule{Id: "rule-1", Filters: []*alertmuting.AlertMutingRuleFilter{nil}})
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "empty filter")
}

func TestResourceAlertMutingRuleMockedLifecycle(t *testing.T) {
	t.Parallel()

	var current alertmuting.AlertMutingRule
	handlers := map[string]http.Handler{
		"POST /v2/alertmuting": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload alertmuting.CreateUpdateAlertMutingRuleRequest
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			assert.Equal(t, int64(1000), payload.StartTime)
			assert.Equal(t, int64(2000), payload.StopTime)
			assert.Equal(t, []string{"detector-1", "detector-2"}, detectorIDs(payload.Filters))
			current = responseFromRequest("rule-1", 123456, payload)
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"GET /v2/alertmuting/rule-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"PUT /v2/alertmuting/rule-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload alertmuting.CreateUpdateAlertMutingRuleRequest
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			assert.Equal(t, int64(123456), payload.StartTime)
			assert.Equal(t, int64(3000), payload.StopTime)
			assert.Equal(t, "updated maintenance", payload.Description)
			assert.Equal(t, []string{"detector-1", "detector-2"}, detectorIDs(payload.Filters))
			current = responseFromRequest("rule-1", 123456, payload)
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"DELETE /v2/alertmuting/rule-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_0_0),
		},
		ProtoV5ProviderFactories: fwtest.NewMockProto5Server(t, handlers, fwtest.WithMockResources(NewResourceAlertMutingRule)),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/00_alert_muting_rule.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_alert_muting_rule.test", "id", "rule-1"),
					testresource.TestCheckResourceAttr("signalfx_alert_muting_rule.test", "effective_start_time", "123456"),
					testresource.TestCheckResourceAttr("signalfx_alert_muting_rule.test", "filter.0.negated", "false"),
				),
			},
			{
				ResourceName:            "signalfx_alert_muting_rule.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"start_time"},
			},
			{
				ConfigFile: config.StaticFile("testdata/01_alert_muting_rule_updated.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_alert_muting_rule.test", "description", "updated maintenance"),
					testresource.TestCheckResourceAttr("signalfx_alert_muting_rule.test", "stop_time", "3"),
				),
			},
		},
	})
}

func TestResourceAlertMutingRuleSchemaValidation(t *testing.T) {
	t.Parallel()

	for name, cfg := range map[string]string{
		"requires detector or filter": `
resource "signalfx_alert_muting_rule" "test" {
  description = "x"
  start_time  = 1
}`,
		"reserved property": `
resource "signalfx_alert_muting_rule" "test" {
  description = "x"
  start_time  = 1
  filter {
    property       = "sf_detectorId"
    property_value = "x"
  }
}`,
		"invalid recurrence": `
resource "signalfx_alert_muting_rule" "test" {
  description = "x"
  start_time  = 1
  detectors   = ["d"]
  recurrence {
    unit  = "m"
    value = 0
  }
}`,
	} {
		t.Run(name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest:               true,
				ProtoV5ProviderFactories: fwtest.NewMockProto5Server(t, nil, fwtest.WithMockResources(NewResourceAlertMutingRule)),
				Steps:                    []testresource.TestStep{{Config: cfg, ExpectError: regexp.MustCompile("(?i)(at least one|none of|invalid|must be)")}},
			})
		})
	}
}

func TestResourceAlertMutingRuleExpiredDelete(t *testing.T) {
	t.Parallel()

	var rule alertmuting.AlertMutingRule
	handlers := map[string]http.Handler{
		"POST /v2/alertmuting": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload alertmuting.CreateUpdateAlertMutingRuleRequest
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			rule = responseFromRequest("rule-1", 123456, payload)
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(rule))
		}),
		"GET /v2/alertmuting/rule-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(rule))
		}),
		"DELETE /v2/alertmuting/rule-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Cannot delete alert muting in the past", http.StatusBadRequest)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: fwtest.NewMockProto5Server(t, handlers, fwtest.WithMockResources(NewResourceAlertMutingRule)),
		Steps: []testresource.TestStep{{
			ConfigFile: config.StaticFile("testdata/00_alert_muting_rule.tf"),
		}},
	})
}

func responseFromRequest(id string, effectiveStart int64, payload alertmuting.CreateUpdateAlertMutingRuleRequest) alertmuting.AlertMutingRule {
	return alertmuting.AlertMutingRule{
		Id:          id,
		Description: payload.Description,
		Filters:     payload.Filters,
		Recurrence:  payload.Recurrence,
		StartTime:   effectiveStart,
		StopTime:    payload.StopTime,
	}
}

func detectorIDs(filters []*alertmuting.AlertMutingRuleFilter) []string {
	var detectors []string
	for _, filter := range filters {
		if filter.Property == alertMutingDetectorIDProperty {
			detectors = append(detectors, filter.PropertyValue.Values...)
		}
	}
	return detectors
}
