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

func TestResourceAzureMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceAzure()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_azure_integration", metadata.TypeName)
	modelWithoutLegacyBlocks := struct {
		integrationModel
		Environment           types.String `tfsdk:"environment"`
		AppID                 types.String `tfsdk:"app_id"`
		SecretKey             types.String `tfsdk:"secret_key"`
		PollRate              types.Int64  `tfsdk:"poll_rate"`
		Services              types.Set    `tfsdk:"services"`
		AdditionalServices    types.List   `tfsdk:"additional_services"`
		Subscriptions         types.Set    `tfsdk:"subscriptions"`
		SyncGuestOSNamespaces types.Bool   `tfsdk:"sync_guest_os_namespaces"`
		ImportAzureMonitor    types.Bool   `tfsdk:"import_azure_monitor"`
		TenantID              types.String `tfsdk:"tenant_id"`
		NamedToken            types.String `tfsdk:"named_token"`
		UseBatchAPI           types.Bool   `tfsdk:"use_batch_api"`
	}{
		Services:           types.SetValueMust(types.StringType, nil),
		AdditionalServices: types.ListValueMust(types.StringType, nil),
		Subscriptions:      types.SetValueMust(types.StringType, nil),
	}
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, modelWithoutLegacyBlocks))

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	assert.True(t, resp.Schema.Attributes["app_id"].IsRequired())
	assert.True(t, resp.Schema.Attributes["app_id"].IsSensitive())
	assert.True(t, resp.Schema.Attributes["secret_key"].IsSensitive())
	assert.True(t, resp.Schema.Attributes["environment"].IsSensitive())
	assert.True(t, resp.Schema.Attributes["services"].IsRequired())
	assert.True(t, resp.Schema.Attributes["subscriptions"].IsRequired())
	assert.Len(t, resp.Schema.Attributes["environment"].(schema.StringAttribute).Validators, 1)
	assert.Len(t, resp.Schema.Attributes["poll_rate"].(schema.Int64Attribute).Validators, 1)
	assert.Len(t, resp.Schema.Attributes["named_token"].(schema.StringAttribute).PlanModifiers, 1)
	_, ok := resp.Schema.Blocks["custom_namespaces_per_service"].(schema.SetNestedBlock)
	assert.True(t, ok, "custom namespaces must preserve set block syntax")
	_, ok = resp.Schema.Blocks["resource_filter_rules"].(schema.ListNestedBlock)
	assert.True(t, ok, "filter rules must preserve ordered block syntax")
}

func TestResourceAzureModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	namespaces, diags := types.SetValueFrom(ctx, types.StringType, []string{"namespace-b", "namespace-a"})
	require.False(t, diags.HasError())
	custom, diags := types.SetValueFrom(ctx, azureCustomNamespaceObjectType, []azureCustomNamespaceModel{{
		Service: types.StringValue("service-a"), Namespaces: namespaces,
	}})
	require.False(t, diags.HasError())
	rules, diags := types.ListValueFrom(ctx, azureFilterRuleObjectType, []azureResourceFilterRuleModel{{
		FilterSource: types.StringValue("filter('azure_tag_env', 'prod')"),
	}})
	require.False(t, diags.HasError())
	services, diags := types.SetValueFrom(ctx, types.StringType, []string{"service-b", "service-a"})
	require.False(t, diags.HasError())
	additional, diags := types.ListValueFrom(ctx, types.StringType, []string{"additional-a", "additional-b"})
	require.False(t, diags.HasError())
	subscriptions, diags := types.SetValueFrom(ctx, types.StringType, []string{"subscription-b", "subscription-a"})
	require.False(t, diags.HasError())

	model := resourceAzureModel{
		integrationModel: integrationModel{ID: types.StringValue("azure-id"), Name: types.StringValue("Azure"), Enabled: types.BoolValue(true)},
		Environment:      types.StringValue(azureEnvironmentUSGovernment), AppID: types.StringValue("app"), SecretKey: types.StringValue("secret"),
		PollRate: types.Int64Value(120), Services: services, AdditionalServices: additional,
		CustomNamespacesPerService: custom, ResourceFilterRules: rules, Subscriptions: subscriptions,
		SyncGuestOSNamespaces: types.BoolValue(true), ImportAzureMonitor: types.BoolValue(false),
		TenantID: types.StringValue("tenant"), NamedToken: types.StringValue("token"), UseBatchAPI: types.BoolValue(true),
	}
	payload, payloadDiags := model.azureIntegration(ctx)
	require.False(t, payloadDiags.HasError())
	assert.Equal(t, integration.Type("Azure"), payload.Type)
	assert.Equal(t, integration.AzureEnvironment("AZURE_US_GOVERNMENT"), payload.AzureEnvironment)
	assert.Equal(t, int64(120000), payload.PollRateMs)
	assert.ElementsMatch(t, []integration.AzureService{"service-a", "service-b"}, payload.Services)
	assert.Equal(t, []string{"additional-a", "additional-b"}, payload.AdditionalServices)
	assert.ElementsMatch(t, []string{"subscription-a", "subscription-b"}, payload.Subscriptions)
	assert.ElementsMatch(t, []string{"namespace-a", "namespace-b"}, payload.CustomNamespacesPerService["service-a"])
	assert.Equal(t, "filter('azure_tag_env', 'prod')", payload.ResourceFilterRules[0].Filter.Source)
	assert.False(t, *payload.ImportAzureMonitor)
	assert.True(t, *payload.UseBatchApi)

	model.updateFromAPI(ctx, nil, true)
	importMonitor := true
	useBatch := false
	readDiags := model.updateFromAPI(ctx, &integration.AzureIntegration{
		Id: "ignored", Name: "Read Azure", Enabled: false, AzureEnvironment: integration.AZURE_DEFAULT,
		PollRateMs: 300000, TenantId: "read-tenant", NamedToken: "read-token",
		Services: []integration.AzureService{"service-a"}, AdditionalServices: []string{"read-additional"},
		Subscriptions: []string{"read-subscription"}, CustomNamespacesPerService: map[string][]string{
			"service-z": {"namespace-z"}, "service-a": {"namespace-b", "namespace-a"},
		},
		ResourceFilterRules:   []integration.AzureFilterRule{{Filter: integration.AzureFilterExpression{Source: "read-filter"}}},
		SyncGuestOsNamespaces: false, ImportAzureMonitor: &importMonitor, UseBatchApi: &useBatch,
	}, false)
	require.False(t, readDiags.HasError())
	assert.Equal(t, types.StringValue("azure-id"), model.ID)
	assert.Equal(t, types.StringValue("app"), model.AppID, "API-omitted application ID must survive refresh")
	assert.Equal(t, types.StringValue("secret"), model.SecretKey, "API-omitted secret must survive refresh")
	assert.Equal(t, types.StringValue(azureEnvironmentDefault), model.Environment)
	assert.Equal(t, types.Int64Value(300), model.PollRate)
	assert.True(t, model.ImportAzureMonitor.ValueBool())
	assert.False(t, model.UseBatchAPI.ValueBool())
	model.Environment = types.StringValue("AZURE")
	readDiags = model.updateFromAPI(ctx, &integration.AzureIntegration{AzureEnvironment: integration.AZURE_DEFAULT}, false)
	require.False(t, readDiags.HasError())
	assert.Equal(t, types.StringValue("AZURE"), model.Environment, "case-insensitive legacy configuration must remain stable")

	model.UseBatchAPI = types.BoolNull()
	model.ImportAzureMonitor = types.BoolValue(true)
	readDiags = model.updateFromAPI(ctx, &integration.AzureIntegration{Id: "updated", Name: "Updated", Enabled: true}, true)
	require.False(t, readDiags.HasError())
	assert.Equal(t, types.StringValue("updated"), model.ID)
	assert.True(t, model.UseBatchAPI.IsNull(), "omitted optional false must remain null")
	assert.True(t, model.ImportAzureMonitor.ValueBool(), "omitted pointer must preserve state")
}

func TestResourceAzureEmptyOptionalCollections(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	services, diags := types.SetValueFrom(ctx, types.StringType, []string{"service"})
	require.False(t, diags.HasError())
	model := resourceAzureModel{
		integrationModel: integrationModel{Name: types.StringValue("Azure"), Enabled: types.BoolValue(true)},
		Environment:      types.StringValue(azureEnvironmentDefault), AppID: types.StringValue("app"), SecretKey: types.StringValue("secret"),
		PollRate: types.Int64Value(300), Services: services, AdditionalServices: types.ListNull(types.StringType),
		CustomNamespacesPerService: types.SetNull(azureCustomNamespaceObjectType),
		ResourceFilterRules:        types.ListNull(azureFilterRuleObjectType), Subscriptions: types.SetNull(types.StringType),
		SyncGuestOSNamespaces: types.BoolValue(false), ImportAzureMonitor: types.BoolValue(true),
		TenantID: types.StringValue("tenant"), NamedToken: types.StringNull(), UseBatchAPI: types.BoolNull(),
	}
	payload, payloadDiags := model.azureIntegration(ctx)
	require.False(t, payloadDiags.HasError())
	assert.Nil(t, payload.AdditionalServices)
	assert.Nil(t, payload.CustomNamespacesPerService)
	assert.Nil(t, payload.ResourceFilterRules)
	assert.Nil(t, payload.Subscriptions)

	beforeServices := model.Services
	beforeAdditional := model.AdditionalServices
	readDiags := model.updateFromAPI(ctx, &integration.AzureIntegration{Name: "Azure", Enabled: true}, false)
	require.False(t, readDiags.HasError())
	assert.Equal(t, beforeServices, model.Services)
	assert.Equal(t, beforeAdditional, model.AdditionalServices)
}

func TestResourceAzureRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceAzure{}
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

func TestResourceAzureMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	importMonitor := true
	useBatch := false
	current := integration.AzureIntegration{
		Id: "azure-id", Name: "Primary Azure", Enabled: true, Type: integration.Type("Azure"),
		AzureEnvironment: integration.AZURE_DEFAULT, PollRateMs: 120000, TenantId: "primary-tenant", NamedToken: "ingest-token",
		Services: []integration.AzureService{"microsoft.compute/virtualmachines"}, Subscriptions: []string{"subscription-a"},
		CustomNamespacesPerService: map[string][]string{"Microsoft.Compute/virtualMachines": {"monitoringAgent", "customNamespace"}},
		ImportAzureMonitor:         &importMonitor, UseBatchApi: &useBatch,
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeAzureResponse(w, current); err != nil {
			t.Errorf("write Azure response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeAzurePayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.Type("Azure"), payload.Type)
			assert.Equal(t, integration.AZURE_DEFAULT, payload.AzureEnvironment)
			assert.Equal(t, "primary-app", payload.AppId)
			assert.Equal(t, "primary-secret", payload.SecretKey)
			assert.Equal(t, int64(120000), payload.PollRateMs)
			writeCurrent(w)
		}),
		"GET /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeAzurePayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated Azure", payload.Name)
			assert.Equal(t, integration.AZURE_US_GOVERNMENT, payload.AzureEnvironment)
			assert.Equal(t, "updated-app", payload.AppId)
			assert.Equal(t, "updated-secret", payload.SecretKey)
			assert.False(t, *payload.ImportAzureMonitor)
			assert.True(t, *payload.UseBatchApi)
			assert.Len(t, payload.ResourceFilterRules, 2)
			mu.Lock()
			current.Name, current.Enabled = payload.Name, payload.Enabled
			current.AzureEnvironment, current.PollRateMs = payload.AzureEnvironment, payload.PollRateMs
			current.TenantId, current.NamedToken = payload.TenantId, payload.NamedToken
			current.Services, current.AdditionalServices = payload.Services, payload.AdditionalServices
			current.Subscriptions, current.CustomNamespacesPerService = payload.Subscriptions, payload.CustomNamespacesPerService
			current.ResourceFilterRules = payload.ResourceFilterRules
			current.SyncGuestOsNamespaces = payload.SyncGuestOsNamespaces
			current.ImportAzureMonitor, current.UseBatchApi = payload.ImportAzureMonitor, payload.UseBatchApi
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAzure)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/azure_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "id", "azure-id"),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "environment", azureEnvironmentDefault),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "app_id", "primary-app"),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "secret_key", "primary-secret"),
			)},
			{ConfigFile: config.StaticFile("testdata/azure_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "name", "Updated Azure"),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "environment", azureEnvironmentUSGovernment),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "additional_services.#", "2"),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "resource_filter_rules.#", "2"),
				testresource.TestCheckResourceAttr("signalfx_azure_integration.test", "use_batch_api", "true"),
			)},
			{ConfigFile: config.StaticFile("testdata/azure_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_azure_integration.test", ImportState: true, ImportStateId: "azure-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"app_id", "secret_key"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceAzureValidation(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceAzure)),
		Steps: []testresource.TestStep{{
			ConfigFile:  config.StaticFile("testdata/azure_invalid.tf"),
			ExpectError: regexp.MustCompile(`(?s)(public|30|at least 1 element)`),
		}},
	})
}

func TestResourceAzureRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	importMonitor := true
	useBatch := false
	current := integration.AzureIntegration{
		Id: "azure-id", Name: "Primary Azure", Enabled: true, Type: integration.Type("Azure"), AzureEnvironment: integration.AZURE_DEFAULT,
		PollRateMs: 120000, TenantId: "primary-tenant", Services: []integration.AzureService{"microsoft.compute/virtualmachines"},
		Subscriptions: []string{"subscription-a"}, ImportAzureMonitor: &importMonitor, UseBatchApi: &useBatch,
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeAzureResponse(w, current); err != nil {
				t.Errorf("write Azure response: %v", err)
			}
		}),
		"GET /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				if err := writeAzureResponse(w, current); err != nil {
					t.Errorf("write Azure response: %v", err)
				}
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAzure)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/azure_create.tf")},
			{ConfigFile: config.StaticFile("testdata/azure_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceAzureErrorHandling(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAzure)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/azure_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceAzureUpdateAdminTokenGuidance(t *testing.T) {
	importMonitor := true
	useBatch := false
	current := integration.AzureIntegration{
		Id: "azure-id", Name: "Primary Azure", Enabled: true, Type: integration.Type("Azure"), AzureEnvironment: integration.AZURE_DEFAULT,
		PollRateMs: 120000, TenantId: "primary-tenant", NamedToken: "ingest-token",
		Services: []integration.AzureService{"microsoft.compute/virtualmachines"}, Subscriptions: []string{"subscription-a"},
		ImportAzureMonitor: &importMonitor, UseBatchApi: &useBatch,
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeAzureResponse(w, current); err != nil {
				t.Errorf("write Azure response: %v", err)
			}
		}),
		"GET /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeAzureResponse(w, current); err != nil {
				t.Errorf("write Azure response: %v", err)
			}
		}),
		"PUT /v2/integration/azure-id":    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "unauthorized", http.StatusUnauthorized) }),
		"DELETE /v2/integration/azure-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAzure)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/azure_create.tf")},
			{ConfigFile: config.StaticFile("testdata/azure_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)},
		},
	})
}

func decodeAzurePayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.AzureIntegration, bool) {
	t.Helper()
	var payload integration.AzureIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode Azure payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeAzureResponse(w http.ResponseWriter, details integration.AzureIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type,
		"azureEnvironment": details.AzureEnvironment, "pollRate": details.PollRateMs,
		"tenantId": details.TenantId, "namedToken": details.NamedToken,
		"services": details.Services, "additionalServices": details.AdditionalServices,
		"subscriptions": details.Subscriptions, "customNamespacesPerService": details.CustomNamespacesPerService,
		"resourceFilterRules": details.ResourceFilterRules, "syncGuestOsNamespaces": details.SyncGuestOsNamespaces,
		"importAzureMonitor": details.ImportAzureMonitor, "useBatchApi": details.UseBatchApi,
	})
}
