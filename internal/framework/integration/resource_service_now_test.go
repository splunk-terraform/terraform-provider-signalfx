// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

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
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceServiceNowMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceServiceNow()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_service_now_integration", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, resourceServiceNowModel{}))

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	assert.True(t, resp.Schema.Attributes["username"].IsRequired())
	assert.True(t, resp.Schema.Attributes["password"].IsRequired())
	assert.True(t, resp.Schema.Attributes["password"].IsSensitive())
	assert.True(t, resp.Schema.Attributes["instance_name"].IsRequired())
	assert.True(t, resp.Schema.Attributes["issue_type"].IsRequired())
	issueType, ok := resp.Schema.Attributes["issue_type"].(schema.StringAttribute)
	require.True(t, ok)
	assert.Len(t, issueType.Validators, 1)
	assert.True(t, resp.Schema.Attributes["alert_triggered_payload_template"].IsOptional())
	assert.True(t, resp.Schema.Attributes["alert_resolved_payload_template"].IsOptional())
}

func TestResourceServiceNowModel(t *testing.T) {
	t.Parallel()
	model := resourceServiceNowModel{
		integrationModel: integrationModel{
			ID: types.StringValue("service-now-id"), Name: types.StringValue("ServiceNow"), Enabled: types.BoolValue(true),
		},
		Username: types.StringValue("primary-user"), Password: types.StringValue("primary-password"),
		InstanceName: types.StringValue("primary.service-now.com"), IssueType: types.StringValue(serviceNowTypeIncident),
		AlertTriggeredPayloadTemplate: types.StringValue(`{"short_description":"primary"}`),
		AlertResolvedPayloadTemplate:  types.StringValue(`{"close_notes":"primary"}`),
	}
	assert.Equal(t, &integration.ServiceNowIntegration{
		Type: integration.SERVICE_NOW, Name: "ServiceNow", Enabled: true,
		Username: "primary-user", Password: "primary-password", InstanceName: "primary.service-now.com", IssueType: serviceNowTypeIncident,
		AlertTriggeredPayloadTemplate: `{"short_description":"primary"}`,
		AlertResolvedPayloadTemplate:  `{"close_notes":"primary"}`,
	}, model.serviceNowIntegration())

	model.updateFromAPI(nil, true)
	model.updateFromAPI(&integration.ServiceNowIntegration{
		Id: "ignored", Name: "Read", Enabled: false, InstanceName: "read.service-now.com", IssueType: serviceNowTypeProblem,
	}, false)
	assert.Equal(t, types.StringValue("service-now-id"), model.ID)
	assert.Equal(t, types.StringValue("primary-user"), model.Username, "API-omitted username must survive refresh")
	assert.Equal(t, types.StringValue("primary-password"), model.Password, "API-omitted password must survive refresh")
	assert.Equal(t, types.StringValue(`{"short_description":"primary"}`), model.AlertTriggeredPayloadTemplate)
	assert.Equal(t, types.StringValue("read.service-now.com"), model.InstanceName)
	assert.Equal(t, types.StringValue(serviceNowTypeProblem), model.IssueType)

	model.updateFromAPI(&integration.ServiceNowIntegration{
		Id: "updated", Name: "Updated", Enabled: true, Username: "api-user", Password: "api-password",
		InstanceName: "updated.service-now.com", IssueType: serviceNowTypeIncident,
		AlertTriggeredPayloadTemplate: "triggered", AlertResolvedPayloadTemplate: "resolved",
	}, true)
	assert.Equal(t, types.StringValue("updated"), model.ID)
	assert.Equal(t, types.StringValue("api-user"), model.Username)
	assert.Equal(t, types.StringValue("api-password"), model.Password)
	assert.Equal(t, types.StringValue("triggered"), model.AlertTriggeredPayloadTemplate)
	assert.Equal(t, types.StringValue("resolved"), model.AlertResolvedPayloadTemplate)
}

func TestResourceServiceNowRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceServiceNow{}
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

func TestResourceServiceNowMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.ServiceNowIntegration{
		Id: "service-now-id", Name: "Primary ServiceNow", Enabled: true, Type: integration.SERVICE_NOW,
		InstanceName: "primary.service-now.com", IssueType: serviceNowTypeIncident,
		AlertTriggeredPayloadTemplate: `{"short_description":"primary"}`,
		AlertResolvedPayloadTemplate:  `{"close_notes":"primary"}`,
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeServiceNowResponse(w, current); err != nil {
			t.Errorf("write ServiceNow response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeServiceNowPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.SERVICE_NOW, payload.Type)
			assert.Equal(t, "primary-user", payload.Username)
			assert.Equal(t, "primary-password", payload.Password)
			assert.Equal(t, serviceNowTypeIncident, payload.IssueType)
			writeCurrent(w)
		}),
		"GET /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeServiceNowPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated ServiceNow", payload.Name)
			assert.False(t, payload.Enabled)
			assert.Equal(t, "updated-user", payload.Username)
			assert.Equal(t, "updated-password", payload.Password)
			assert.Equal(t, serviceNowTypeProblem, payload.IssueType)
			mu.Lock()
			current.Name, current.Enabled = payload.Name, payload.Enabled
			current.InstanceName, current.IssueType = payload.InstanceName, payload.IssueType
			current.AlertTriggeredPayloadTemplate = payload.AlertTriggeredPayloadTemplate
			current.AlertResolvedPayloadTemplate = payload.AlertResolvedPayloadTemplate
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceServiceNow)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/service_now_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "id", "service-now-id"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "username", "primary-user"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "password", "primary-password"),
			)},
			{ConfigFile: config.StaticFile("testdata/service_now_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "name", "Updated ServiceNow"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "enabled", "false"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "username", "updated-user"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "password", "updated-password"),
				testresource.TestCheckResourceAttr("signalfx_service_now_integration.test", "issue_type", serviceNowTypeProblem),
			)},
			{ConfigFile: config.StaticFile("testdata/service_now_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_service_now_integration.test", ImportState: true, ImportStateId: "service-now-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"username", "password"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceServiceNowValidation(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceServiceNow)),
		Steps: []testresource.TestStep{{
			ConfigFile:  config.StaticFile("testdata/service_now_invalid.tf"),
			ExpectError: regexp.MustCompile("Change"),
		}},
	})
}

func TestResourceServiceNowRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.ServiceNowIntegration{
		Id: "service-now-id", Name: "Primary ServiceNow", Enabled: true, Type: integration.SERVICE_NOW,
		InstanceName: "primary.service-now.com", IssueType: serviceNowTypeIncident,
		AlertTriggeredPayloadTemplate: `{"short_description":"primary"}`,
		AlertResolvedPayloadTemplate:  `{"close_notes":"primary"}`,
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := writeServiceNowResponse(w, current); err != nil {
			t.Errorf("write ServiceNow response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				writeCurrent(w)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceServiceNow)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/service_now_create.tf")},
			{ConfigFile: config.StaticFile("testdata/service_now_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceServiceNowErrorHandling(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		error  *regexp.Regexp
	}{
		{name: "administrator guidance", status: http.StatusUnauthorized, error: regexp.MustCompile(adminTokenHelp)},
		{name: "server failure", status: http.StatusBadGateway, error: regexp.MustCompile(`status code 502`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoints := map[string]http.Handler{"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", test.status) })}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceServiceNow)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/service_now_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceServiceNowUpdateAdminTokenGuidance(t *testing.T) {
	current := integration.ServiceNowIntegration{
		Id: "service-now-id", Name: "Primary ServiceNow", Enabled: true, Type: integration.SERVICE_NOW,
		InstanceName: "primary.service-now.com", IssueType: serviceNowTypeIncident,
		AlertTriggeredPayloadTemplate: `{"short_description":"primary"}`,
		AlertResolvedPayloadTemplate:  `{"close_notes":"primary"}`,
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := writeServiceNowResponse(w, current); err != nil {
			t.Errorf("write ServiceNow response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration":               http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}),
		"DELETE /v2/integration/service-now-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceServiceNow)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/service_now_create.tf")},
			{ConfigFile: config.StaticFile("testdata/service_now_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)},
		},
	})
}

func decodeServiceNowPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.ServiceNowIntegration, bool) {
	t.Helper()
	var payload integration.ServiceNowIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode ServiceNow payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeServiceNowResponse(w http.ResponseWriter, details integration.ServiceNowIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type,
		"instanceName": details.InstanceName, "issueType": details.IssueType,
		"alertTriggeredPayloadTemplate": details.AlertTriggeredPayloadTemplate,
		"alertResolvedPayloadTemplate":  details.AlertResolvedPayloadTemplate,
	})
}
