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

func TestResourceOpsgenieMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceOpsgenie()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_opsgenie_integration", resp.TypeName)
}

func TestResourceOpsgenieSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceOpsgenie(), resourceOpsgenieModel{}))
}

func TestResourceOpsgenieModel(t *testing.T) {
	t.Parallel()

	model := resourceOpsgenieModel{
		ID:      types.StringValue("integration-id"),
		Name:    types.StringValue("Opsgenie"),
		Enabled: types.BoolValue(true),
		APIKey:  types.StringValue("secret-key"),
		APIURL:  types.StringValue("https://api.opsgenie.com"),
	}

	assert.Equal(t, &integration.OpsgenieIntegration{
		Type:    "Opsgenie",
		Name:    "Opsgenie",
		Enabled: true,
		ApiKey:  "secret-key",
		ApiUrl:  "https://api.opsgenie.com",
	}, model.opsgenieIntegration())

	model.updateFromAPI(nil)
	assert.Equal(t, types.StringValue("integration-id"), model.ID)

	model.updateFromAPI(&integration.OpsgenieIntegration{
		Id:      "updated-id",
		Name:    "Updated Opsgenie",
		Enabled: false,
	})
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated Opsgenie"), model.Name)
	assert.Equal(t, types.BoolValue(false), model.Enabled)
	assert.Equal(t, types.StringValue("secret-key"), model.APIKey, "API key must survive API reads")
	assert.Equal(t, types.StringValue("https://api.opsgenie.com"), model.APIURL, "API URL must survive API reads")
}

func TestResourceOpsgenieRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	implementation := &ResourceOpsgenie{}
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

func TestResourceOpsgenieMockedLifecycle(t *testing.T) {
	var (
		mu      sync.Mutex
		current = integration.OpsgenieIntegration{
			Id:      "opsgenie-id",
			Name:    "Primary Opsgenie",
			Enabled: true,
			Type:    "Opsgenie",
		}
		created bool
		deleted bool
	)

	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeOpsgenieResponse(w, current); err != nil {
			t.Errorf("write Opsgenie response: %v", err)
		}
	}

	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.OpsgenieIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode create payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.Type("Opsgenie"), payload.Type)
			assert.Equal(t, "Primary Opsgenie", payload.Name)
			assert.True(t, payload.Enabled)
			assert.Equal(t, "secret-key", payload.ApiKey)
			assert.Equal(t, "https://api.opsgenie.com", payload.ApiUrl)

			mu.Lock()
			created = true
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"GET /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCurrent(w)
		}),
		"PUT /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.OpsgenieIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode update payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.Type("Opsgenie"), payload.Type)
			assert.Equal(t, "Updated Opsgenie", payload.Name)
			assert.False(t, payload.Enabled)
			assert.Equal(t, "updated-secret-key", payload.ApiKey)
			assert.Equal(t, "https://api.eu.opsgenie.com", payload.ApiUrl)

			mu.Lock()
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			fwtest.WithMockResources(NewResourceOpsgenie),
		),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/opsgenie_create.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "id", "opsgenie-id"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "name", "Primary Opsgenie"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "enabled", "true"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "api_key", "secret-key"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "api_url", "https://api.opsgenie.com"),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/opsgenie_update.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "id", "opsgenie-id"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "name", "Updated Opsgenie"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "enabled", "false"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "api_key", "updated-secret-key"),
					testresource.TestCheckResourceAttr("signalfx_opsgenie_integration.test", "api_url", "https://api.eu.opsgenie.com"),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/opsgenie_update.tf"),
				PlanOnly:   true,
			},
			{
				ResourceName:            "signalfx_opsgenie_integration.test",
				ImportState:             true,
				ImportStateId:           "opsgenie-id",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key", "api_url"},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, created)
	assert.True(t, deleted)
}

func TestResourceOpsgenieRemovesMissingState(t *testing.T) {
	var getCalls int
	var mu sync.Mutex

	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := writeOpsgenieResponse(w, integration.OpsgenieIntegration{
				Id:      "opsgenie-id",
				Name:    "Primary Opsgenie",
				Enabled: true,
			}); err != nil {
				t.Errorf("write create response: %v", err)
			}
		}),
		"GET /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				if err := writeOpsgenieResponse(w, integration.OpsgenieIntegration{
					Id:      "opsgenie-id",
					Name:    "Primary Opsgenie",
					Enabled: true,
				}); err != nil {
					t.Errorf("write read response: %v", err)
				}
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			endpoints,
			fwtest.WithMockResources(NewResourceOpsgenie),
		),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/opsgenie_create.tf")},
			{
				ConfigFile:         config.StaticFile("testdata/opsgenie_create.tf"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestResourceOpsgenieAdminTokenGuidance(t *testing.T) {
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
		},
		{
			name: "update",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if err := writeOpsgenieResponse(w, integration.OpsgenieIntegration{
						Id:      "opsgenie-id",
						Name:    "Primary Opsgenie",
						Enabled: true,
					}); err != nil {
						t.Errorf("write create response: %v", err)
					}
				}),
				"GET /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if err := writeOpsgenieResponse(w, integration.OpsgenieIntegration{
						Id:      "opsgenie-id",
						Name:    "Primary Opsgenie",
						Enabled: true,
					}); err != nil {
						t.Errorf("write read response: %v", err)
					}
				}),
				"PUT /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
				}),
				"DELETE /v2/integration/opsgenie-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			steps: []testresource.TestStep{
				{ConfigFile: config.StaticFile("testdata/opsgenie_create.tf")},
				{
					ConfigFile:  config.StaticFile("testdata/opsgenie_update.tf"),
					ExpectError: regexp.MustCompile(adminTokenHelp),
				},
			},
		},
	} {
		steps := test.steps
		if len(steps) == 0 {
			steps = []testresource.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/opsgenie_create.tf"),
					ExpectError: regexp.MustCompile(adminTokenHelp),
				},
			}
		}

		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t,
					test.endpoints,
					fwtest.WithMockResources(NewResourceOpsgenie),
				),
				Steps: steps,
			})
		})
	}
}

func TestResourceOpsgenieRejectsMissingAPIKey(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			nil,
			fwtest.WithMockResources(NewResourceOpsgenie),
		),
		Steps: []testresource.TestStep{
			{
				Config: `
resource "signalfx_opsgenie_integration" "test" {
  name    = "Missing key"
  enabled = true
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`The argument "api_key" is required`),
			},
		},
	})
}

func writeOpsgenieResponse(w http.ResponseWriter, details integration.OpsgenieIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id":      details.Id,
		"name":    details.Name,
		"enabled": details.Enabled,
		"type":    details.Type,
	})
}
