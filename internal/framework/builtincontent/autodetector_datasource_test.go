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
	"github.com/signalfx/signalfx-go/detector"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestAutoDetectorMetadata(t *testing.T) {
	t.Parallel()

	ad := NewAutoDetectorDataSource()
	var resp datasource.MetadataResponse
	ad.Metadata(t.Context(), datasource.MetadataRequest{ProviderTypeName: "signalfx"}, &resp)

	assert.Equal(t, "signalfx_auto_detector", resp.TypeName, "Must match the expected name")
}

func TestAutoDetectorSchema(t *testing.T) {
	t.Parallel()

	ad := NewAutoDetectorDataSource()
	var resp datasource.SchemaResponse
	ad.Schema(t.Context(), datasource.SchemaRequest{}, &resp)

	assert.NotEmpty(t, resp.Schema.Description, "Must have a description set")
	assert.NotEmpty(t, resp.Schema.Attributes, "Must have values defined for attributes")
}

func TestAutoDetectorMockIntegration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		steps     []resourcetest.TestStep
	}{
		{
			name: "detector endpoint returns error",
			endpoints: map[string]http.Handler{
				"GET /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()
					http.Error(w, "Not Serving Requests", http.StatusBadGateway)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/auto-detector.tf"),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`route "/v2/detector" had issues with status code 502`),
				},
			},
		},
		{
			name: "successfully loads auto detectors",
			endpoints: map[string]http.Handler{
				"GET /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					searched := &detector.SearchResults{
						Count: 3,
						Results: []detector.Detector{
							{
								Id:             "detector-1",
								Name:           "CPU Utilization",
								DetectorOrigin: "AutoDetect",
							},
							{
								Id:             "detector-2",
								Name:           "Manual Detector",
								DetectorOrigin: "Standard",
							},
							{
								Id:             "detector-3",
								Name:           "Disk Errors (%)",
								DetectorOrigin: "AutoDetect",
							},
						},
					}
					if err := json.NewEncoder(w).Encode(searched); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					}
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/auto-detector.tf"),
					PlanOnly:   true,
					Check: resourcetest.ComposeTestCheckFunc(
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.%", "2"),
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.CPU_Utilization", "detector-1"),
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.Disk_Errors", "detector-3"),
					),
				},
			},
		},
		{
			name: "loads all detector pages",
			endpoints: map[string]http.Handler{
				"GET /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					var searched *detector.SearchResults
					switch r.URL.Query().Get("offset") {
					case "0":
						searched = &detector.SearchResults{
							Count: 100,
							Results: []detector.Detector{
								{
									Id:             "detector-1",
									Name:           "First Auto Detector",
									DetectorOrigin: "AutoDetect",
								},
							},
						}
						for idx := 2; idx <= 100; idx++ {
							searched.Results = append(searched.Results, detector.Detector{
								Id:             "standard-detector",
								Name:           "Standard Detector",
								DetectorOrigin: "Standard",
							})
						}
					case "100":
						searched = &detector.SearchResults{
							Count: 1,
							Results: []detector.Detector{
								{
									Id:             "detector-101",
									Name:           "Second Auto Detector",
									DetectorOrigin: "AutoDetect",
								},
							},
						}
					default:
						t.Fatalf("unexpected offset %q", r.URL.Query().Get("offset"))
					}

					if err := json.NewEncoder(w).Encode(searched); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					}
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/auto-detector.tf"),
					PlanOnly:   true,
					Check: resourcetest.ComposeTestCheckFunc(
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.%", "2"),
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.First_Auto_Detector", "detector-1"),
						resourcetest.TestCheckResourceAttr("data.signalfx_auto_detector.test", "results.Second_Auto_Detector", "detector-101"),
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
					fwtest.WithMockDataSources(NewAutoDetectorDataSource),
				),
				Steps: tc.steps,
			})
		})
	}
}
