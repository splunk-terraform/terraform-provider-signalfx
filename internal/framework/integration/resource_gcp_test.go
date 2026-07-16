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

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

func TestResourceGCPMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceGCP()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_gcp_integration", metadata.TypeName)

	modelWithoutLegacyBlocks := struct {
		integrationModel
		PollRate                       types.Int64  `tfsdk:"poll_rate"`
		Services                       types.Set    `tfsdk:"services"`
		CustomMetricTypeDomains        types.Set    `tfsdk:"custom_metric_type_domains"`
		AuthMethod                     types.String `tfsdk:"auth_method"`
		WorkloadIdentityConfig         types.String `tfsdk:"workload_identity_federation_config"`
		WIFSplunkIdentity              types.Map    `tfsdk:"wif_splunk_identity"`
		UseMetricSourceProjectForQuota types.Bool   `tfsdk:"use_metric_source_project_for_quota"`
		IncludeList                    types.Set    `tfsdk:"include_list"`
		NamedToken                     types.String `tfsdk:"named_token"`
		ImportGCPMetrics               types.Bool   `tfsdk:"import_gcp_metrics"`
		ExcludeGCEInstancesWithLabels  types.Set    `tfsdk:"exclude_gce_instances_with_labels"`
	}{
		Services: types.SetValueMust(types.StringType, nil), CustomMetricTypeDomains: types.SetValueMust(types.StringType, nil),
		WIFSplunkIdentity: types.MapValueMust(types.StringType, nil), IncludeList: types.SetValueMust(types.StringType, nil),
		ExcludeGCEInstancesWithLabels: types.SetValueMust(types.StringType, nil),
	}
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, modelWithoutLegacyBlocks))

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	auth := resp.Schema.Attributes["auth_method"].(schema.StringAttribute)
	assert.True(t, auth.IsOptional())
	assert.True(t, auth.IsComputed())
	assert.Len(t, auth.Validators, 1)
	assert.Len(t, resp.Schema.Attributes["poll_rate"].(schema.Int64Attribute).Validators, 1)
	assert.Len(t, resp.Schema.Attributes["named_token"].(schema.StringAttribute).PlanModifiers, 1)
	serviceKeys := resp.Schema.Blocks["project_service_keys"].(schema.SetNestedBlock)
	assert.Len(t, serviceKeys.Validators, 1)
	assert.True(t, serviceKeys.NestedObject.Attributes["project_id"].IsSensitive())
	assert.True(t, serviceKeys.NestedObject.Attributes["project_key"].IsSensitive())
	_, hasDeprecatedWIFConfigs := resp.Schema.Blocks["project_wif_configs"]
	assert.False(t, hasDeprecatedWIFConfigs)
	projects := resp.Schema.Blocks["projects"].(schema.ListNestedBlock)
	assert.Len(t, projects.Validators, 1)
	assert.Len(t, projects.NestedObject.Attributes["sync_mode"].(schema.StringAttribute).Validators, 1)
}

func TestResourceGCPModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	serviceKeys := gcpProjectServiceKeys(t, []gcpProjectServiceKeyModel{
		{ProjectID: types.StringValue("project-a"), ProjectKey: types.StringValue("secret-a")},
		{ProjectID: types.StringValue("project-b"), ProjectKey: types.StringValue("secret-b")},
	})
	projects := gcpProjects(t, gcpSyncSelected, []string{"project-b", "project-a"})
	model := resourceGCPModel{
		integrationModel: integrationModel{ID: types.StringValue("gcp-id"), Name: types.StringValue("GCP"), Enabled: types.BoolValue(true)},
		PollRate:         types.Int64Value(300), Services: gcpStrings(t, "storage", "compute"),
		CustomMetricTypeDomains: gcpStrings(t, "custom.googleapis.com"), AuthMethod: types.StringValue("workload_identity_federation"),
		ProjectServiceKeys:     serviceKeys,
		WorkloadIdentityConfig: types.StringValue("federation-json"), Projects: projects,
		WIFSplunkIdentity: types.MapNull(types.StringType), UseMetricSourceProjectForQuota: types.BoolValue(true),
		IncludeList: gcpStrings(t, "labels"), NamedToken: types.StringValue("token"), ImportGCPMetrics: types.BoolValue(false),
		ExcludeGCEInstancesWithLabels: gcpStrings(t, "test"),
	}
	payload, diags := model.gcpIntegration(ctx)
	require.False(t, diags.HasError())
	assert.Equal(t, integration.Type("GCP"), payload.Type)
	assert.Equal(t, integration.WORKLOAD_IDENTITY_FEDERATION, payload.AuthMethod)
	assert.Equal(t, int64(300000), payload.PollRateMs)
	assert.ElementsMatch(t, []integration.GcpService{"compute", "storage"}, payload.Services)
	assert.Len(t, payload.ProjectServiceKeys, 2)
	assert.Equal(t, "secret-a", payload.ProjectServiceKeys[0].ProjectKey)
	assert.Empty(t, payload.WifConfigs)
	assert.ElementsMatch(t, []string{"project-a", "project-b"}, payload.Projects.SelectedProjectIds)
	assert.False(t, *payload.ImportGCPMetrics)

	model.updateFromAPI(ctx, nil, true)
	importMetrics := true
	readDiags := model.updateFromAPI(ctx, &integration.GCPIntegration{
		Id: "ignored", Name: "Read", Enabled: false, PollRateMs: 60000, AuthMethod: integration.WORKLOAD_IDENTITY_FEDERATION,
		Services: []integration.GcpService{"compute"}, CustomMetricTypeDomains: []string{"read.googleapis.com"},
		WorkloadIdentityFederationConfig: "read-federation", Projects: &integration.GCPProjects{SyncMode: integration.ALL_REACHABLE},
		WifSplunkIdentity: map[string]string{"account_id": "account"}, UseMetricSourceProjectForQuota: false,
		IncludeList: []string{"metadata"}, ImportGCPMetrics: &importMetrics,
		ExcludeGCEInstancesWithLabels: []string{"read-label"},
	}, false)
	require.False(t, readDiags.HasError())
	assert.Equal(t, types.StringValue("gcp-id"), model.ID)
	assert.Equal(t, serviceKeys, model.ProjectServiceKeys, "API-omitted service account keys must survive refresh")
	assert.Equal(t, types.Int64Value(60), model.PollRate)
	assert.Equal(t, types.StringValue("workload_identity_federation"), model.AuthMethod, "case-insensitive configuration must remain stable")
	assert.Equal(t, "account", model.WIFSplunkIdentity.Elements()["account_id"].(types.String).ValueString())

	model.WorkloadIdentityConfig = types.StringValue("old")
	model.WIFSplunkIdentity = types.MapValueMust(types.StringType, map[string]attr.Value{"old": types.StringValue("old")})
	clearDiags := model.updateFromAPI(ctx, &integration.GCPIntegration{Id: "updated", Name: "Updated", Enabled: true}, true)
	require.False(t, clearDiags.HasError())
	assert.Equal(t, types.StringValue("updated"), model.ID)
	assert.True(t, model.WorkloadIdentityConfig.IsNull())
	assert.True(t, model.WIFSplunkIdentity.IsNull())
}

func TestResourceGCPStringSetHelpers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	value := gcpStrings(t, "b", "a")
	values, diags := stringSetElements(ctx, value)
	require.False(t, diags.HasError())
	assert.ElementsMatch(t, []string{"a", "b"}, values)
	values, diags = stringSetElements(ctx, types.SetNull(types.StringType))
	assert.Nil(t, values)
	assert.Empty(t, diags)
	target := types.SetNull(types.StringType)
	assert.Empty(t, updateStringSet(ctx, &target, nil, true))
	assert.True(t, target.IsNull())
	assert.Empty(t, updateStringSet(ctx, &target, []string{"value"}, true))
	assert.Len(t, target.Elements(), 1)
	assert.Empty(t, updateStringSet(ctx, &target, nil, true))
	assert.Empty(t, target.Elements())
	assert.False(t, target.IsNull())
}

func TestResourceGCPRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceGCP{}
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

func TestResourceGCPMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	importMetrics := true
	current := integration.GCPIntegration{
		Id: "gcp-id", Name: "Primary GCP", Enabled: true, Type: integration.Type("GCP"), PollRateMs: 600000,
		Services: []integration.GcpService{"compute"}, CustomMetricTypeDomains: []string{"custom.googleapis.com"},
		AuthMethod: integration.SERVICE_ACCOUNT_KEY, IncludeList: []string{"labels"}, NamedToken: "ingest-token",
		ImportGCPMetrics: &importMetrics, ExcludeGCEInstancesWithLabels: []string{"development", "test"},
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeGCPResponse(w, current); err != nil {
			t.Errorf("write GCP response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeGCPPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.Type("GCP"), payload.Type)
			assert.Len(t, payload.ProjectServiceKeys, 2)
			assert.ElementsMatch(t, []string{"secret-a", "secret-b"}, []string{payload.ProjectServiceKeys[0].ProjectKey, payload.ProjectServiceKeys[1].ProjectKey})
			writeCurrent(w)
		}),
		"GET /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeGCPPayload(t, w, r)
			if !ok {
				return
			}
			mu.Lock()
			current.Name, current.Enabled, current.PollRateMs = payload.Name, payload.Enabled, payload.PollRateMs
			current.Services, current.CustomMetricTypeDomains = payload.Services, payload.CustomMetricTypeDomains
			current.AuthMethod, current.WifConfigs = payload.AuthMethod, payload.WifConfigs
			current.WorkloadIdentityFederationConfig, current.Projects = payload.WorkloadIdentityFederationConfig, payload.Projects
			current.UseMetricSourceProjectForQuota, current.IncludeList = payload.UseMetricSourceProjectForQuota, payload.IncludeList
			current.NamedToken, current.ImportGCPMetrics = payload.NamedToken, payload.ImportGCPMetrics
			current.ExcludeGCEInstancesWithLabels = payload.ExcludeGCEInstancesWithLabels
			if payload.Projects != nil {
				current.WifSplunkIdentity = map[string]string{"account_id": "splunk-account"}
			}
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceGCP)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/gcp_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "id", "gcp-id"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "project_service_keys.#", "2"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "auth_method", gcpAuthServiceAccount),
			)},
			{ConfigFile: config.StaticFile("testdata/gcp_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "name", "Updated GCP"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "projects.#", "1"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "import_gcp_metrics", "false"),
			)},
			{ConfigFile: config.StaticFile("testdata/gcp_modern_wif.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "name", "Modern WIF GCP"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "projects.#", "1"),
				testresource.TestCheckResourceAttr("signalfx_gcp_integration.test", "wif_splunk_identity.account_id", "splunk-account"),
			)},
			{ConfigFile: config.StaticFile("testdata/gcp_modern_wif.tf"), PlanOnly: true},
			{ResourceName: "signalfx_gcp_integration.test", ImportState: true, ImportStateId: "gcp-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"project_service_keys"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceGCPValidation(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceGCP)),
		Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/gcp_invalid.tf"), ExpectError: regexp.MustCompile(`(?s)(PASSWORD|30|conflict|at most 1)`)}},
	})
}

func TestResourceGCPRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	importMetrics := true
	current := integration.GCPIntegration{
		Id: "gcp-id", Name: "Primary GCP", Enabled: true, Type: integration.Type("GCP"), PollRateMs: 600000,
		Services: []integration.GcpService{"compute"}, CustomMetricTypeDomains: []string{"custom.googleapis.com"},
		AuthMethod: integration.SERVICE_ACCOUNT_KEY, IncludeList: []string{"labels"}, ImportGCPMetrics: &importMetrics,
		ExcludeGCEInstancesWithLabels: []string{"development", "test"}, NamedToken: "ingest-token",
	}
	write := func(w http.ResponseWriter) {
		if err := writeGCPResponse(w, current); err != nil {
			t.Errorf("write GCP response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { write(w) }),
		"GET /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				write(w)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceGCP)),
		Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/gcp_create.tf")}, {ConfigFile: config.StaticFile("testdata/gcp_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true}},
	})
}

func TestResourceGCPErrorHandling(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceGCP)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/gcp_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceGCPUpdateAdminTokenGuidance(t *testing.T) {
	importMetrics := true
	current := integration.GCPIntegration{
		Id: "gcp-id", Name: "Primary GCP", Enabled: true, Type: integration.Type("GCP"), PollRateMs: 600000,
		Services: []integration.GcpService{"compute"}, CustomMetricTypeDomains: []string{"custom.googleapis.com"},
		AuthMethod: integration.SERVICE_ACCOUNT_KEY, IncludeList: []string{"labels"}, NamedToken: "ingest-token",
		ImportGCPMetrics: &importMetrics, ExcludeGCEInstancesWithLabels: []string{"development", "test"},
	}
	write := func(w http.ResponseWriter) {
		if err := writeGCPResponse(w, current); err != nil {
			t.Errorf("write GCP response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration":          http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { write(w) }),
		"GET /v2/integration/gcp-id":    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { write(w) }),
		"PUT /v2/integration/gcp-id":    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "unauthorized", http.StatusUnauthorized) }),
		"DELETE /v2/integration/gcp-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceGCP)),
		Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/gcp_create.tf")}, {ConfigFile: config.StaticFile("testdata/gcp_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)}},
	})
}

func gcpStrings(t *testing.T, values ...string) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError())
	return value
}

func gcpProjectServiceKeys(t *testing.T, values []gcpProjectServiceKeyModel) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), gcpProjectServiceKeyObjectType, values)
	require.False(t, diags.HasError())
	return value
}

func gcpProjects(t *testing.T, syncMode string, selected []string) types.List {
	t.Helper()
	value, diags := types.ListValueFrom(context.Background(), gcpProjectsObjectType, []gcpProjectsModel{{
		SyncMode: types.StringValue(syncMode), SelectedProjectIDs: gcpStrings(t, selected...),
	}})
	require.False(t, diags.HasError())
	return value
}

func decodeGCPPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.GCPIntegration, bool) {
	t.Helper()
	var payload integration.GCPIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode GCP payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeGCPResponse(w http.ResponseWriter, details integration.GCPIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type,
		"pollRate": details.PollRateMs, "services": details.Services,
		"customMetricTypeDomains": details.CustomMetricTypeDomains, "authMethod": details.AuthMethod,
		"workloadIdentityFederationConfigs": details.WifConfigs, "wifSplunkIdentity": details.WifSplunkIdentity,
		"workloadIdentityFederationConfig": details.WorkloadIdentityFederationConfig, "projects": details.Projects,
		"useMetricSourceProjectForQuota": details.UseMetricSourceProjectForQuota, "includeList": details.IncludeList,
		"namedToken": details.NamedToken, "importGCPMetrics": details.ImportGCPMetrics,
		"excludeGCEInstancesWithLabels": details.ExcludeGCEInstancesWithLabels,
	})
}
