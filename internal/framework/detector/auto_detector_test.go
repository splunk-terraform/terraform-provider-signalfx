// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdetector

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestAutoDetectorResourceMetadata(t *testing.T) {
	t.Parallel()

	r := NewAutoDetectorResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)

	assert.Equal(t, "signalfx_customized_auto_detector", resp.TypeName)
}

func TestAutoDetectorResourceSchema(t *testing.T) {
	t.Parallel()

	r := NewAutoDetectorResource()
	resp := &resource.SchemaResponse{}
	r.Schema(t.Context(), resource.SchemaRequest{}, resp)

	assert.NotEmpty(t, resp.Schema.Description)
	assert.NotEmpty(t, resp.Schema.Attributes)
	assert.Contains(t, resp.Schema.Attributes, "parent_id")
	assert.Contains(t, resp.Schema.Attributes, "filters")
}

func TestAutoDetectorResource(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		endpoints map[string]http.Handler
		steps     []resourcetest.TestStep
	}{
		{
			name: "parent detector does not exist",
			endpoints: map[string]http.Handler{
				"GET /v2/detector/parent-detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "not found", http.StatusNotFound)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/auto_detect.tf"),
					ExpectError: regexp.MustCompilePOSIX(`Unable to load auto detector`),
				},
			},
		},
		{
			name: "invalid parent detector",
			endpoints: map[string]http.Handler{
				"GET /v2/detector/standard-detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					d := detector.Detector{
						Id:             "standard-detector",
						Name:           "Standard Detector",
						DetectorOrigin: "Detector",
					}
					_ = json.NewEncoder(w).Encode(&d)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/auto_detect_invalid_parent.tf"),
					ExpectError: regexp.MustCompilePOSIX(`Invalid Parent ID`),
				},
			},
		},
		{
			name: "invalid input",
			endpoints: map[string]http.Handler{
				"GET /v2/detector/parent-detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					d := detector.Detector{
						Id:             "parent-detector",
						Name:           "Example #01",
						DetectorOrigin: "AutoDetect",
					}
					_ = json.NewEncoder(w).Encode(&d)
				}),
				"POST /v2/signalflow/_/getSignalFlowModel": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					content := `[
			{"alias": null,"module": "signalfx.detectors.autodetect.apm","name": "requests","type": "IMPORT"},
			{"start": {"functionName": "requests.blended", "originalText": "requests.blended().publish('Request Rate Dropped')","position": 56,"type": "user_function"},"streamMethods": [{"args": {"label": {"originalText": "'Request Rate Dropped'","position": 83,"type": "string","value": "Request Rate Dropped"}},"functionName": "publish","originalText": "requests.blended().publish('Request Rate Dropped')","position": 56,"type": "stream_method"}],"type": "DETECT","uniqueKey": 0}
			]`
					_, _ = w.Write([]byte(content))
				}),
				"GET /v2/signalflow/_/extractMetadata": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					content := metadata.Metadata{
						Arguments: []metadata.Argument{
							{
								Name:        "guardrail",
								Type:        "string",
								Label:       "Argument 1",
								Description: "This is argument 1",
							},
						},
					}
					_ = json.NewEncoder(w).Encode(&content)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile:  config.StaticFile("testdata/auto_detect_invalid_input.tf"),
					ExpectError: regexp.MustCompilePOSIX(`Invalid Input`),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resourcetest.UnitTest(t, resourcetest.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, tc.endpoints, fwtest.WithMockResources(NewAutoDetectorResource),
				),
				Steps: tc.steps,
			})
		})
	}
}
