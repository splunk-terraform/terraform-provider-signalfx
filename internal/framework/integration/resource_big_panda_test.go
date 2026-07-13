// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceBigPandaMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceBigPanda()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_big_panda_integration", resp.TypeName)
}

func TestResourceBigPandaSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceBigPanda(), resourceBigPandaModel{}))
}

func TestResourceBigPandaUnitTest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		cases     []testresource.TestStep
	}{
		{
			name: "create and update integration",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.BigPandaIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					assert.Equal(t, integration.BIG_PANDA, data.Type)
					assert.Equal(t, "BigPanda - My Team", data.Name)
					assert.True(t, data.Enabled)
					assert.Equal(t, "my-app-key", data.AppKey)
					assert.Equal(t, "my-token", data.Token)
					assert.Empty(t, data.AlertTriggeredPayloadTemplate)
					assert.Empty(t, data.AlertResolvedPayloadTemplate)

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}),
				"GET /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data := integration.BigPandaIntegration{
						Id:                            "test-id",
						Type:                          integration.BIG_PANDA,
						Name:                          "BigPanda - My Team",
						Enabled:                       true,
						AlertTriggeredPayloadTemplate: "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}",
						AlertResolvedPayloadTemplate:  "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}",
					}
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}),
				"PUT /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.BigPandaIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					assert.Equal(t, integration.BIG_PANDA, data.Type)
					assert.Equal(t, "BigPanda - My Team", data.Name)
					assert.False(t, data.Enabled)
					assert.JSONEq(t, "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}", data.AlertTriggeredPayloadTemplate)
					assert.JSONEq(t, "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}", data.AlertResolvedPayloadTemplate)

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}),
				"DELETE /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_big_panda.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "enabled", "true"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "name", "BigPanda - My Team"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "app_key", "my-app-key"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "token", "my-token"),
					),
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectUnknownValue("signalfx_big_panda_integration.test", tfjsonpath.New("id")),
							plancheck.ExpectKnownValue("signalfx_big_panda_integration.test", tfjsonpath.New("name"), knownvalue.StringExact("BigPanda - My Team")),
							plancheck.ExpectKnownValue("signalfx_big_panda_integration.test", tfjsonpath.New("enabled"), knownvalue.Bool(true)),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_big_panda_integration.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
						},
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/01_big_panda_with_payloads.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "enabled", "false"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "name", "BigPanda - My Team"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "alert_triggered_payload_template", "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"),
						testresource.TestCheckResourceAttr("signalfx_big_panda_integration.test", "alert_resolved_payload_template", "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"),
					),
					ExpectNonEmptyPlan: true,
				},
			},
		},
		{
			name: "invalid token",
			endpoints: map[string]http.Handler{
				"/v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile:         config.StaticFile("testdata/00_big_panda.tf"),
					ExpectNonEmptyPlan: false,
					ExpectError:        regexp.MustCompile("route \"/v2/integration\" had issues with status code 401"),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fwtest.UnitTest(
				t,
				testresource.TestCase{
					IsUnitTest: true,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{
						tfversion.RequireAbove(tfversion.Version0_12_26),
					},
					ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
						t,
						tc.endpoints,
						fwtest.WithMockResources(NewResourceBigPanda),
					),
					Steps: tc.cases,
				},
			)
		})
	}
}
