// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestNewDatasourceTopology(t *testing.T) {
	t.Parallel()

	var (
		ds   = NewDatasourceTopology()
		resp = datasource.MetadataResponse{}
	)

	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, &resp)
	assert.Equal(t, "signalfx_apm_service_topology", resp.TypeName)
}

func TestDatasourceTopologySchema(t *testing.T) {
	t.Skip("Needs to include an update in the fwtest to include support for data sources")
}

func TestDatasourceTopology(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		steps     []resourcetest.TestStep
	}{
		{
			name: "validating plan step",
			endpoints: map[string]http.Handler{
				"POST /v2/apm/topology": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					topology := map[string]any{
						"data": map[string]any{
							"nodes": []map[string]any{
								{"serviceName": "my-awesome-service", "inferred": true},
							},
							"edges": []map[string]any{
								{"fromNode": "service-a", "toNode": "service-b"},
							},
						},
					}
					if err := json.NewEncoder(w).Encode(topology); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/service_topology_example.tf"),
					ConfigPlanChecks: resourcetest.ConfigPlanChecks{
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectKnownOutputValue("nodes", knownvalue.NotNull()),
							plancheck.ExpectKnownOutputValue("edges", knownvalue.NotNull()),
						},
					},
				},
			},
		},
		{
			name: "invalid mux of filters must fail",
			endpoints: map[string]http.Handler{
				"POST /v2/apm/topology": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					topology := map[string]any{
						"data": map[string]any{
							"nodes": []map[string]any{
								{"serviceName": "my-awesome-service", "inferred": true},
							},
						},
					}
					if err := json.NewEncoder(w).Encode(topology); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/service_topology_misuse.tf"),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
				},
			},
		},
		{
			name: "Too small of lookback period",
			endpoints: map[string]http.Handler{
				"POST /v2/apm/topology": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Should not be called", http.StatusBadRequest)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/service_topology_bad_lookback.tf"),
					ConfigPlanChecks: resourcetest.ConfigPlanChecks{
						PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectNonEmptyPlan(),
						},
					},
					ExpectError: regexp.MustCompile("Error: Invalid Time Range"),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resourcetest.UnitTest(
				t,
				resourcetest.TestCase{
					ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
						t,
						tc.endpoints,
						fwtest.WithMockDataSources(NewDatasourceTopology),
					),
					Steps: tc.steps,
				},
			)
		})
	}
}
