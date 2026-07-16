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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceVictorOpsMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceVictorOps()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_victor_ops_integration", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, resourceVictorOpsModel{}))

	schemaResponse := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	postURL := schemaResponse.Schema.Attributes["post_url"]
	assert.True(t, postURL.IsOptional(), "legacy post_url is optional")
	assert.False(t, postURL.IsSensitive(), "legacy post_url is not sensitive")
}

func TestResourceVictorOpsModel(t *testing.T) {
	t.Parallel()
	model := resourceVictorOpsModel{
		integrationModel: integrationModel{
			ID: types.StringValue("victor-ops-id"), Name: types.StringValue("VictorOps"), Enabled: types.BoolValue(true),
		},
		PostURL: types.StringValue("https://alert.victorops.test/secret"),
	}
	assert.Equal(t, &integration.VictorOpsIntegration{
		Type: integration.VICTOR_OPS, Name: "VictorOps", Enabled: true, PostUrl: "https://alert.victorops.test/secret",
	}, model.victorOpsIntegration())
	model.updateFromAPI(nil, true)
	model.updateFromAPI(&integration.VictorOpsIntegration{Id: "ignored", Name: "Read", Enabled: false}, false)
	assert.Equal(t, types.StringValue("victor-ops-id"), model.ID)
	assert.Equal(t, types.StringValue("https://alert.victorops.test/secret"), model.PostURL)
	model.updateFromAPI(&integration.VictorOpsIntegration{Id: "updated", Name: "Updated", Enabled: true}, true)
	assert.Equal(t, types.StringValue("updated"), model.ID)
}

func TestResourceVictorOpsRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceVictorOps{}
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

func TestResourceVictorOpsMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.VictorOpsIntegration{Id: "victor-ops-id", Name: "Primary VictorOps", Enabled: true, Type: integration.VICTOR_OPS}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeVictorOpsResponse(w, current); err != nil {
			t.Errorf("write VictorOps response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeVictorOpsPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.VICTOR_OPS, payload.Type)
			assert.Equal(t, "Primary VictorOps", payload.Name)
			assert.True(t, payload.Enabled)
			assert.Equal(t, "https://alert.victorops.test/primary", payload.PostUrl)
			writeCurrent(w)
		}),
		"GET /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeVictorOpsPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated VictorOps", payload.Name)
			assert.False(t, payload.Enabled)
			assert.Equal(t, "https://alert.victorops.test/updated", payload.PostUrl)
			mu.Lock()
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceVictorOps)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/victor_ops_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_victor_ops_integration.test", "id", "victor-ops-id"),
				testresource.TestCheckResourceAttr("signalfx_victor_ops_integration.test", "post_url", "https://alert.victorops.test/primary"),
			)},
			{ConfigFile: config.StaticFile("testdata/victor_ops_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_victor_ops_integration.test", "name", "Updated VictorOps"),
				testresource.TestCheckResourceAttr("signalfx_victor_ops_integration.test", "enabled", "false"),
				testresource.TestCheckResourceAttr("signalfx_victor_ops_integration.test", "post_url", "https://alert.victorops.test/updated"),
			)},
			{ConfigFile: config.StaticFile("testdata/victor_ops_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_victor_ops_integration.test", ImportState: true, ImportStateId: "victor-ops-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"post_url"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceVictorOpsRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.VictorOpsIntegration{Id: "victor-ops-id", Name: "Primary VictorOps", Enabled: true, Type: integration.VICTOR_OPS}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeVictorOpsResponse(w, current); err != nil {
				t.Errorf("write create response: %v", err)
			}
		}),
		"GET /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				if err := writeVictorOpsResponse(w, current); err != nil {
					t.Errorf("write read response: %v", err)
				}
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceVictorOps)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/victor_ops_create.tf")},
			{ConfigFile: config.StaticFile("testdata/victor_ops_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceVictorOpsErrorHandling(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceVictorOps)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/victor_ops_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceVictorOpsUpdateAdminTokenGuidance(t *testing.T) {
	current := integration.VictorOpsIntegration{Id: "victor-ops-id", Name: "Primary VictorOps", Enabled: true, Type: integration.VICTOR_OPS}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeVictorOpsResponse(w, current); err != nil {
				t.Errorf("write create response: %v", err)
			}
		}),
		"GET /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeVictorOpsResponse(w, current); err != nil {
				t.Errorf("write read response: %v", err)
			}
		}),
		"PUT /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}),
		"DELETE /v2/integration/victor-ops-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceVictorOps)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/victor_ops_create.tf")},
			{ConfigFile: config.StaticFile("testdata/victor_ops_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)},
		},
	})
}

func decodeVictorOpsPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.VictorOpsIntegration, bool) {
	t.Helper()
	var payload integration.VictorOpsIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode VictorOps payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeVictorOpsResponse(w http.ResponseWriter, details integration.VictorOpsIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type})
}
