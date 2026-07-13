// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package builtincontent

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/dashboard"
	"github.com/signalfx/signalfx-go/dashboard_group"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestDashboardGroupsMetadata(t *testing.T) {
	t.Parallel()

	dg := NewDashboardGroupsDataSource()
	var resp datasource.MetadataResponse
	dg.Metadata(t.Context(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, &resp)

	assert.Equal(t, "signalfx_builtin_dashboards", resp.TypeName, "Must match the expected name")
}

func TestDashboardGroupsSchema(t *testing.T) {
	t.Parallel()

	dg := NewDashboardGroupsDataSource()
	var resp datasource.SchemaResponse
	dg.Schema(t.Context(), datasource.SchemaRequest{}, &resp)

	assert.NotEmpty(t, resp.Schema.Description, "Must have a description set")
	assert.NotEmpty(t, resp.Schema.Attributes, "Must have values defined for attributes")
}

func TestDashboardGroupsClean(t *testing.T) {
	t.Parallel()

	dg := NewDashboardGroupsDataSource().(*DashboardGroupsDataSource)

	for _, tc := range []struct {
		name   string
		in     string
		expect string
	}{
		{name: "empty string", in: "", expect: ""},
		{name: "unmodified string", in: "collector", expect: "collector"},
		{name: "string with spaces", in: "Otel Collector", expect: "Otel_Collector"},
		{name: "string with special formatting", in: "Otel Collector (2.x)", expect: "Otel_Collector_2_x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, dg.clean(tc.in), "Must clean the string as expected")
		})
	}
}

func TestDashboardGroupMockIngeration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		steps     []resourcetest.TestStep
	}{
		{
			name: "dashboard group endpoint returns error",
			endpoints: map[string]http.Handler{
				"GET /v2/dashboardgroup": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					http.Error(w, "Not Serving Requests", http.StatusBadGateway)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/builtin-dashboards.tf"),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`route "/v2/dashboardgroup" had issues with status code 502`),
				},
			},
		},
		{
			name: "Unable to resolve dashboard information",
			endpoints: map[string]http.Handler{
				"GET /v2/dashboardgroup": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					searched := &dashboard_group.SearchResult{
						Count: 1,
						Results: []*dashboard_group.DashboardGroup{
							{
								Name:       "Test Dashboard Group",
								Dashboards: []string{"dashboard-1"},
							},
						},
					}
					if err := json.NewEncoder(w).Encode(searched); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					}
				}),
				"GET /v2/dashboard/dashboard-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Not Serving Requests", http.StatusBadGateway)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/builtin-dashboards.tf"),
					PlanOnly:    true,
					ExpectError: regexp.MustCompilePOSIX(`route "/v2/dashboard/dashboard-1" had issues with status code 502`),
				},
			},
		},
		{
			name: "successfully loaded built in content",
			endpoints: map[string]http.Handler{
				"GET /v2/dashboardgroup": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					searched := &dashboard_group.SearchResult{
						Count: 1,
						Results: []*dashboard_group.DashboardGroup{
							{
								Name:       "Test Dashboard Group",
								Dashboards: []string{"dashboard-1"},
							},
						},
					}
					if err := json.NewEncoder(w).Encode(searched); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					}
				}),
				"GET /v2/dashboard/dashboard-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					dashboard := &dashboard.Dashboard{
						Id:   "dashboard-1",
						Name: "Test Dashboard",
					}
					if err := json.NewEncoder(w).Encode(dashboard); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					}
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/builtin-dashboards.tf"),
					PlanOnly:   true,
					Check: resourcetest.ComposeTestCheckFunc(
						resourcetest.TestCheckResourceAttr("signalfx_builtin_dashbiards.example", "results.%", "1"),
						resourcetest.TestCheckResourceAttr("signalfx_builtin_dashbiards.example", "results.Test_Dashboard_Group.%", "1"),
						resourcetest.TestCheckResourceAttr("signalfx_builtin_dashbiards.example", "results.Test_Dashboard_Group.Test_Dashboard", "dashboard-1"),
					),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resourcetest.UnitTest(t, resourcetest.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t,
					tc.endpoints,
					fwtest.WithMockDataSources(NewDashboardGroupsDataSource),
				),
				Steps: tc.steps,
			})
		})
	}
}
