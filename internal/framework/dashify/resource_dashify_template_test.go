// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

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
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceDashifyTemplateMetadata(t *testing.T) {
	t.Parallel()

	r := NewResourceDashifyTemplate()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_dashify_template", resp.TypeName)
}

func TestResourceDashifyTemplateSchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fwtest.ResourceSchemaValidate(NewResourceDashifyTemplate(), resourceDashifyTemplateModel{}))
}

func TestResourceDashifyTemplateUnitTest(t *testing.T) {
	t.Parallel()

	templateCounter := 0
	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		cases     []testresource.TestStep
	}{
		{
			name: "Correctly configured client",
			endpoints: map[string]http.Handler{
				"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					templateCounter++
					var data map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					// Validate required fields
					if _, ok := data["metadata"]; !ok {
						http.Error(w, "metadata is required", http.StatusBadRequest)
						return
					}
					if _, ok := data["spec"]; !ok {
						http.Error(w, "spec is required", http.StatusBadRequest)
						return
					}

					// Create response with API-generated fields
					response := map[string]interface{}{
						"data": map[string]interface{}{
							"id":        "test-template-id",
							"createdAt": "2025-12-04T18:00:00.000Z[UTC]",
							"createdBy": "/v2/user/test-user",
							"self":      "/v2/template/test-template-id",
							"updatedAt": "2025-12-04T18:00:00.000Z[UTC]",
							"updatedBy": "/v2/user/test-user",
							// Include user-provided fields with API normalization
							"metadata": data["metadata"],
							"spec":     data["spec"],
							"title":    data["title"],
							"type":     "https://schema.splunkdev.com/dashify/v1/templates/Record",
						},
						"errors":   []interface{}{},
						"includes": []interface{}{},
					}

					if err := json.NewEncoder(w).Encode(response); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"GET /v2/template/test-template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := map[string]interface{}{
						"data": map[string]interface{}{
							"id":        "test-template-id",
							"createdAt": "2025-12-04T18:00:00.000Z[UTC]",
							"createdBy": "/v2/user/test-user",
							"self":      "/v2/template/test-template-id",
							"updatedAt": "2025-12-04T18:00:00.000Z[UTC]",
							"updatedBy": "/v2/user/test-user",
							"metadata": map[string]interface{}{
								"rootElement": "Chart",
								"imports":     []interface{}{},
							},
							"spec": map[string]interface{}{
								"<Chart>": []interface{}{
									map[string]interface{}{
										"<o11y:TimeSeriesChart>": []interface{}{},
										"chart":                  map[string]interface{}{},
										"datasource": map[string]interface{}{
											"program":    "A = data('cpu.utilization').publish('A')",
											"resolution": 1000,
										},
									},
								},
							},
							"title": "Test Template",
							"type":  "https://schema.splunkdev.com/dashify/v1/templates/Record",
						},
						"errors":   []interface{}{},
						"includes": []interface{}{},
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"PUT /v2/template/test-template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var data map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					response := map[string]interface{}{
						"data": map[string]interface{}{
							"id":        "test-template-id",
							"updatedAt": "2025-12-04T18:01:00.000Z[UTC]",
							"updatedBy": "/v2/user/test-user",
							"metadata":  data["metadata"],
							"spec":      data["spec"],
							"title":     data["title"],
							"type":      "https://schema.splunkdev.com/dashify/v1/templates/Record",
						},
						"errors":   []interface{}{},
						"includes": []interface{}{},
					}

					if err := json.NewEncoder(w).Encode(response); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
				"DELETE /v2/template/test-template-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_dashify_template.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_dashify_template.test", "id", "test-template-id"),
						testresource.TestCheckResourceAttrSet("signalfx_dashify_template.test", "template_contents"),
					),
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectUnknownValue("signalfx_dashify_template.test", tfjsonpath.New("id")),
						},
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_dashify_template.test", tfjsonpath.New("id"), knownvalue.StringExact("test-template-id")),
						},
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/01_updated_template.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_dashify_template.test", "id", "test-template-id"),
						testresource.TestCheckResourceAttrSet("signalfx_dashify_template.test", "template_contents"),
					),
					ExpectNonEmptyPlan: true,
					ConfigPlanChecks: testresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectKnownValue("signalfx_dashify_template.test", tfjsonpath.New("id"), knownvalue.StringExact("test-template-id")),
						},
					},
				},
			},
		},
		{
			name: "Invalid JSON handling",
			endpoints: map[string]http.Handler{
				"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "invalid JSON", http.StatusBadRequest)
				}),
			},
			cases: []testresource.TestStep{
				{
					Config: `
resource "signalfx_dashify_template" "test" {
  template_contents = "invalid json"
}
`,
					ExpectError: regexp.MustCompile("Invalid JSON"),
				},
			},
		},
		{
			name: "API error handling",
			endpoints: map[string]http.Handler{
				"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}),
			},
			cases: []testresource.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/00_dashify_template.tf"),
					ExpectError: regexp.MustCompile("Error Creating Template"),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
					t,
					tc.endpoints,
					fwtest.WithMockResources(NewResourceDashifyTemplate),
				),
				Steps: tc.cases,
			})
		})
	}
}

func TestResourceDashifyTemplateJSONValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name: "Valid JSON",
			json: `{"metadata": {"rootElement": "Chart"}, "spec": {}, "title": "Test"}`,
		},
		{
			name:        "Invalid JSON",
			json:        `{invalid json}`,
			expectError: true,
		},
		{
			name:        "Empty string",
			json:        ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var js interface{}
			err := json.Unmarshal([]byte(tt.json), &js)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
