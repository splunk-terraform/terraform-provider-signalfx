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

func TestResourceSplunkOncallMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceSplunkOncall()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_integration_splunk_oncall", resp.TypeName)
}

func TestResourceSplunkOnCallSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceSplunkOncall(), resourceSplunkOnCallModel{}))
}

func TestResourceSplunkOncallUnitTest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		cases     []testresource.TestStep
	}{
		{
			name: "Correctly configured client",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if data.Name == "" || data.PostUrl == "" {
						http.Error(w, "name and post_url are required", http.StatusBadRequest)
						return
					}

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"GET /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data := integration.VictorOpsIntegration{
						Id:      "test-id",
						Name:    "Test Integration",
						Enabled: true,
						PostUrl: "https://example.com/post",
					}
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"PUT /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"DELETE /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_splunk_oncall.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "true"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/splunk_oncall"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectUnknownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("name"), knownvalue.StringExact("Test Integration")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("enabled"), knownvalue.Bool(true)),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/01_modified_integration.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "false"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/post"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("name"), knownvalue.StringExact("Test Integration")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("enabled"), knownvalue.Bool(false)),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/post")),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/post")),
						},
					},
				},
			},
		},
		{
			name: "invalid token",
			endpoints: map[string]http.Handler{
				"/v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_splunk_oncall.tf"),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: false,
					ExpectError:        regexp.MustCompile("route \"/v2/integration\" had issues with status code 401"),
				},
			},
		},
		{
			name: "fail unable to modify resource",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if data.Name == "" || data.PostUrl == "" {
						http.Error(w, "name and post_url are required", http.StatusBadRequest)
						return
					}

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"GET /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data := integration.VictorOpsIntegration{
						Id:      "test-id",
						Name:    "Test Integration",
						Enabled: true,
						PostUrl: "https://example.com/post",
					}
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"PUT /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					http.Error(w, "Unable to modify resource", http.StatusBadRequest)
				}),
				"DELETE /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_splunk_oncall.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "true"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/splunk_oncall"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectUnknownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("name"), knownvalue.StringExact("Test Integration")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("enabled"), knownvalue.Bool(true)),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/01_modified_integration.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "false"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/post"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: false,
					ExpectError:        regexp.MustCompile("route \"/v2/integration/test-id\" had issues with status code 400"),
				},
			},
		},
		{
			name: "state is cleared on removed resource",
			endpoints: map[string]http.Handler{
				"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if data.Name == "" || data.PostUrl == "" {
						http.Error(w, "name and post_url are required", http.StatusBadRequest)
						return
					}

					data.Id = "test-id"
					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"GET /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					http.Error(w, "Resource not found", http.StatusNotFound)
				}),
				"PUT /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data integration.VictorOpsIntegration
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if err := json.NewEncoder(w).Encode(data); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"DELETE /v2/integration/test-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body) // Drain the body
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_splunk_oncall.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "true"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/splunk_oncall"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("signalfx_integration_splunk_oncall.test", plancheck.ResourceActionCreate),
							plancheck.ExpectUnknownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("name"), knownvalue.StringExact("Test Integration")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("enabled"), knownvalue.Bool(true)),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/splunk_oncall")),
						},
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/01_modified_integration.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "id", "test-id"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "enabled", "false"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "name", "Test Integration"),
						testresource.TestCheckResourceAttr("signalfx_integration_splunk_oncall.test", "post_url", "https://example.com/post"),
					),
					// This will check to see if the resource already exists
					// and cause an update in place so the plan is expected to be non empty.
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("signalfx_integration_splunk_oncall.test", plancheck.ResourceActionCreate),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("id"), knownvalue.StringExact("test-id")),
							plancheck.ExpectKnownValue("signalfx_integration_splunk_oncall.test", tfjsonpath.New("post_url"), knownvalue.StringExact("https://example.com/post")),
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testresource.UnitTest(
				t,
				testresource.TestCase{
					IsUnitTest: true,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{
						tfversion.RequireAbove(tfversion.Version0_12_26),
					},
					ProtoV5ProviderFactories: fwtest.NewMockProviderFactory(
						t,
						tc.endpoints,
						fwtest.WithMockResources(NewResourceSplunkOncall),
					),
					Steps: tc.cases,
				},
			)
		})
	}
}
