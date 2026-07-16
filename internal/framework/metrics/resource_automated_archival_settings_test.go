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

func TestResourceAutomatedArchivalSettingsMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceAutomatedArchivalSettings()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_automated_archival_settings", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(implementation, resourceAutomatedArchivalSettingsModel{}))
}

func TestResourceAutomatedArchivalSettingsModel(t *testing.T) {
	t.Parallel()
	model := resourceAutomatedArchivalSettingsModel{
		ID: types.StringValue("1"), Creator: types.StringValue("creator"),
		LastUpdatedBy: types.StringValue("updater"), Created: types.Int64Value(100), LastUpdated: types.Int64Value(200),
		Version: types.StringValue("2"), Enabled: types.BoolValue(true),
		LookbackPeriod: types.StringValue("P30D"), GracePeriod: types.StringValue("P15D"), RulesetLimit: types.Int32Value(10),
	}
	payload, diagnostics := model.toAPI()
	require.False(t, diagnostics.HasError())
	assert.Equal(t, int64(2), payload.Version)
	assert.Equal(t, "creator", *payload.Creator)
	assert.Equal(t, int32(10), *payload.RulesetLimit)

	creator := "new-creator"
	updated := int64(300)
	limit := int32(20)
	model.updateFromAPI(&automatedarchival.AutomatedArchivalSettings{
		Creator: &creator, LastUpdated: &updated, Version: 3, Enabled: false,
		LookbackPeriod: "P45D", GracePeriod: "P30D", RulesetLimit: &limit,
	}, false)
	assert.Equal(t, types.StringValue("3"), model.Version)
	assert.Equal(t, types.BoolValue(false), model.Enabled)
	assert.Equal(t, types.StringValue("new-creator"), model.Creator)
	assert.Equal(t, types.Int32Value(20), model.RulesetLimit)

	version, err := model.latestVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(3), version)
	model.Version = types.StringNull()
	version, err = model.latestVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)
	model.ID = types.StringValue("invalid")
	_, err = model.latestVersion()
	assert.Error(t, err)
}

func TestResourceAutomatedArchivalSettingsModelUnknowns(t *testing.T) {
	t.Parallel()
	model := resourceAutomatedArchivalSettingsModel{
		Version: types.StringValue("invalid"), Creator: types.StringUnknown(), LastUpdatedBy: types.StringUnknown(),
		Created: types.Int64Unknown(), LastUpdated: types.Int64Unknown(), RulesetLimit: types.Int32Unknown(),
	}
	_, diagnostics := model.toAPI()
	assert.True(t, diagnostics.HasError())
	model.updateFromAPI(&automatedarchival.AutomatedArchivalSettings{}, true)
	assert.True(t, model.Creator.IsNull())
	assert.True(t, model.LastUpdatedBy.IsNull())
	assert.True(t, model.Created.IsNull())
	assert.True(t, model.LastUpdated.IsNull())
	assert.True(t, model.RulesetLimit.IsNull())
	model.updateFromAPI(nil, false)

	prior := resourceAutomatedArchivalSettingsModel{
		ID: types.StringValue("1"), Version: types.StringValue("2"), Creator: types.StringValue("creator"),
		RulesetLimit: types.Int32Value(10),
	}
	model.RulesetLimit = types.Int32Unknown()
	model.copyComputedFrom(prior)
	assert.Equal(t, prior.ID, model.ID)
	assert.Equal(t, prior.RulesetLimit, model.RulesetLimit)
}

func TestResourceAutomatedArchivalSettingsRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceAutomatedArchivalSettings{}
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

func TestResourceAutomatedArchivalSettingsMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := automatedarchival.AutomatedArchivalSettings{}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write settings response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeSettingsPayload(t, w, r)
			if !ok {
				return
			}
			assert.True(t, payload.Enabled)
			assert.Equal(t, int64(0), payload.Version)
			assert.Equal(t, int32(10), *payload.RulesetLimit)
			creator, updater := "creator", "creator"
			created, updated := int64(100), int64(100)
			mu.Lock()
			current = payload
			current.Version, current.Creator, current.LastUpdatedBy = 1, &creator, &updater
			current.Created, current.LastUpdated = &created, &updated
			mu.Unlock()
			writeCurrent(w)
		}),
		"GET /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeSettingsPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, int64(1), payload.Version)
			assert.False(t, payload.Enabled)
			assert.Equal(t, int32(20), *payload.RulesetLimit)
			updater := "updater"
			updated := int64(200)
			mu.Lock()
			current = payload
			current.Version, current.LastUpdatedBy, current.LastUpdated = 2, &updater, &updated
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload automatedarchival.AutomatedArchivalSettingsDeleteRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode delete: %v", err)
				return
			}
			if assert.NotNil(t, payload.Version) {
				assert.Equal(t, int64(2), *payload.Version)
			}
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalSettings)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/automated_archival_settings_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "id", "1"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "version", "1"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "creator", "creator"),
			)},
			{ConfigFile: config.StaticFile("testdata/automated_archival_settings_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "id", "1"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "version", "2"),
				testresource.TestCheckResourceAttr("signalfx_automated_archival_settings.test", "last_updated_by", "updater"),
			)},
			{ConfigFile: config.StaticFile("testdata/automated_archival_settings_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_automated_archival_settings.test", ImportState: true, ImportStateId: "1", ImportStateVerify: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceAutomatedArchivalSettingsRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := automatedarchival.AutomatedArchivalSettings{Version: 1, Enabled: true, LookbackPeriod: "P30D", GracePeriod: "P15D"}
	endpoints := map[string]http.Handler{
		"POST /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(current) }),
		"GET /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				_ = json.NewEncoder(w).Encode(current)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalSettings)),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_automated_archival_settings" "test" {
  enabled = true
  lookback_period = "P30D"
  grace_period = "P15D"
}`},
			{Config: `resource "signalfx_automated_archival_settings" "test" {
  enabled = true
  lookback_period = "P30D"
  grace_period = "P15D"
}`, PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceAutomatedArchivalSettingsErrors(t *testing.T) {
	valid := `resource "signalfx_automated_archival_settings" "test" {
  enabled = true
  lookback_period = "P30D"
  grace_period = "P15D"
}`
	for _, test := range []struct {
		name, config string
		endpoints    map[string]http.Handler
		error        *regexp.Regexp
	}{
		{name: "API error", config: valid, endpoints: map[string]http.Handler{"POST /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", http.StatusBadGateway) })}, error: regexp.MustCompile(`status code 502`)},
		{name: "nil API response", config: valid, endpoints: map[string]http.Handler{"POST /v2/automated-archival/settings": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`null`)) })}, error: regexp.MustCompile(`returned no settings`)},
		{name: "missing enabled", config: `resource "signalfx_automated_archival_settings" "test" {
  lookback_period = "P30D"
  grace_period = "P15D"
}`, error: regexp.MustCompile(`argument "enabled" is required`)},
		{name: "ruleset overflow", config: `resource "signalfx_automated_archival_settings" "test" {
  enabled = true
  lookback_period = "P30D"
  grace_period = "P15D"
  ruleset_limit = 2147483648
}`, error: regexp.MustCompile(`32-bit[[:space:]]+integer`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockResources(NewResourceAutomatedArchivalSettings)), Steps: []testresource.TestStep{{Config: test.config, ExpectError: test.error}}})
		})
	}
}

func decodeSettingsPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (automatedarchival.AutomatedArchivalSettings, bool) {
	t.Helper()
	var payload automatedarchival.AutomatedArchivalSettings
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode settings payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}
