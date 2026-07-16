// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"io"
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

func TestResourcePagerDutyMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourcePagerDuty()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_pagerduty_integration", resp.TypeName)
}

func TestResourcePagerDutySchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourcePagerDuty(), resourcePagerDutyModel{}))

	resp := &resource.SchemaResponse{}
	NewResourcePagerDuty().Schema(context.Background(), resource.SchemaRequest{}, resp)
	apiKey := resp.Schema.Attributes["api_key"]
	assert.True(t, apiKey.IsOptional(), "legacy api_key is optional")
	assert.True(t, apiKey.IsSensitive())
}

func TestResourcePagerDutyModel(t *testing.T) {
	t.Parallel()

	model := resourcePagerDutyModel{
		integrationModel: integrationModel{
			ID:      types.StringValue("pagerduty-id"),
			Name:    types.StringValue("PagerDuty"),
			Enabled: types.BoolValue(true),
		},
		APIKey: types.StringValue("pagerduty-key"),
	}

	assert.Equal(t, &integration.PagerDutyIntegration{
		Type:    "PagerDuty",
		Name:    "PagerDuty",
		Enabled: true,
		ApiKey:  "pagerduty-key",
	}, model.pagerDutyIntegration())

	model.updateFromAPI(nil, true)
	assert.Equal(t, types.StringValue("pagerduty-id"), model.ID)

	model.updateFromAPI(&integration.PagerDutyIntegration{
		Id:      "ignored-id",
		Name:    "Updated PagerDuty",
		Enabled: false,
	}, false)
	assert.Equal(t, types.StringValue("pagerduty-id"), model.ID, "refresh must retain the state ID")
	assert.Equal(t, types.StringValue("Updated PagerDuty"), model.Name)
	assert.Equal(t, types.BoolValue(false), model.Enabled)
	assert.Equal(t, types.StringValue("pagerduty-key"), model.APIKey, "API key must survive API reads")

	model.updateFromAPI(&integration.PagerDutyIntegration{
		Id:      "created-id",
		Name:    "Created PagerDuty",
		Enabled: true,
	}, true)
	assert.Equal(t, types.StringValue("created-id"), model.ID)
}

func TestResourcePagerDutyRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	implementation := &ResourcePagerDuty{}
	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	invalidPlan := tfsdk.Plan{
		Raw:    tftypes.NewValue(tftypes.Bool, true),
		Schema: schemaResponse.Schema,
	}
	invalidState := tfsdk.State{
		Raw:    tftypes.NewValue(tftypes.Bool, true),
		Schema: schemaResponse.Schema,
	}

	createResponse := &resource.CreateResponse{}
	implementation.Create(ctx, resource.CreateRequest{Plan: invalidPlan}, createResponse)
	assert.True(t, createResponse.Diagnostics.HasError())

	readResponse := &resource.ReadResponse{}
	implementation.Read(ctx, resource.ReadRequest{State: invalidState}, readResponse)
	assert.True(t, readResponse.Diagnostics.HasError())

	updateResponse := &resource.UpdateResponse{}
	implementation.Update(ctx, resource.UpdateRequest{Plan: invalidPlan}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())

	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: invalidState}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourcePagerDutyMockedLifecycle(t *testing.T) {
	var (
		mu      sync.Mutex
		current = integration.PagerDutyIntegration{
			Id:      "pagerduty-id",
			Name:    "Primary PagerDuty",
			Enabled: true,
			Type:    "PagerDuty",
		}
		created bool
		deleted bool
	)

	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writePagerDutyResponse(w, current); err != nil {
			t.Errorf("write PagerDuty response: %v", err)
		}
	}

	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodePagerDutyPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.Type("PagerDuty"), payload.Type)
			assert.Equal(t, "Primary PagerDuty", payload.Name)
			assert.True(t, payload.Enabled)
			assert.Equal(t, "pagerduty-key", payload.ApiKey)

			mu.Lock()
			created = true
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"GET /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCurrent(w)
		}),
		"PUT /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodePagerDutyPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated PagerDuty", payload.Name)
			assert.False(t, payload.Enabled)
			assert.Equal(t, "updated-pagerduty-key", payload.ApiKey)

			mu.Lock()
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			endpoints,
			fwtest.WithMockResources(NewResourcePagerDuty),
		),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pagerduty_create.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "id", "pagerduty-id"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "name", "Primary PagerDuty"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "enabled", "true"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "api_key", "pagerduty-key"),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/pagerduty_update.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "id", "pagerduty-id"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "name", "Updated PagerDuty"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "enabled", "false"),
					testresource.TestCheckResourceAttr("signalfx_pagerduty_integration.test", "api_key", "updated-pagerduty-key"),
				),
			},
			{ConfigFile: config.StaticFile("testdata/pagerduty_update.tf"), PlanOnly: true},
			{
				ResourceName:            "signalfx_pagerduty_integration.test",
				ImportState:             true,
				ImportStateId:           "pagerduty-id",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, created)
	assert.True(t, deleted)
}

func TestResourcePagerDutyRemovesMissingState(t *testing.T) {
	var getCalls int
	var mu sync.Mutex

	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writePagerDutyResponse(w, integration.PagerDutyIntegration{
				Id: "pagerduty-id", Name: "Primary PagerDuty", Enabled: true,
			}); err != nil {
				t.Errorf("write create response: %v", err)
			}
		}),
		"GET /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				if err := writePagerDutyResponse(w, integration.PagerDutyIntegration{
					Id: "pagerduty-id", Name: "Primary PagerDuty", Enabled: true,
				}); err != nil {
					t.Errorf("write read response: %v", err)
				}
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, endpoints, fwtest.WithMockResources(NewResourcePagerDuty),
		),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/pagerduty_create.tf")},
			{
				ConfigFile:         config.StaticFile("testdata/pagerduty_create.tf"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestResourcePagerDutyAdminTokenGuidance(t *testing.T) {
	for _, test := range []struct {
		name      string
		endpoints map[string]http.Handler
		steps     []testresource.TestStep
	}{
		{
			name: "create",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
				}),
			},
			steps: []testresource.TestStep{{
				ConfigFile:  config.StaticFile("testdata/pagerduty_create.tf"),
				ExpectError: regexp.MustCompile(adminTokenHelp),
			}},
		},
		{
			name: "update",
			endpoints: map[string]http.Handler{
				"POST /v2/integration":             pagerDutyResponseHandler(t, "pagerduty-id", "Primary PagerDuty", true),
				"GET /v2/integration/pagerduty-id": pagerDutyResponseHandler(t, "pagerduty-id", "Primary PagerDuty", true),
				"PUT /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
				}),
				"DELETE /v2/integration/pagerduty-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			steps: []testresource.TestStep{
				{ConfigFile: config.StaticFile("testdata/pagerduty_create.tf")},
				{
					ConfigFile:  config.StaticFile("testdata/pagerduty_update.tf"),
					ExpectError: regexp.MustCompile(adminTokenHelp),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, test.endpoints, fwtest.WithMockResources(NewResourcePagerDuty),
				),
				Steps: test.steps,
			})
		})
	}
}

func decodePagerDutyPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.PagerDutyIntegration, bool) {
	t.Helper()

	var payload integration.PagerDutyIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode PagerDuty payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return integration.PagerDutyIntegration{}, false
	}
	return payload, true
}

func writePagerDutyResponse(w http.ResponseWriter, details integration.PagerDutyIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id":      details.Id,
		"name":    details.Name,
		"enabled": details.Enabled,
		"type":    details.Type,
	})
}

func pagerDutyResponseHandler(t *testing.T, id, name string, enabled bool) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := writePagerDutyResponse(w, integration.PagerDutyIntegration{
			Id: id, Name: name, Enabled: enabled, Type: "PagerDuty",
		}); err != nil {
			t.Errorf("write PagerDuty response: %v", err)
		}
	})
}
