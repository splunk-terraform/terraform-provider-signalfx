// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdetector

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
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
		{
			name: "valid lifecycle",
			endpoints: map[string]http.Handler{
				"GET /v2/detector/parent-detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					det := detector.Detector{
						Id:             "parent-detector",
						Name:           "Example #01",
						DetectorOrigin: "AutoDetect",
						ProgramText: "from signalfx.detectors.autodetect.apm import requests \n" +
							`requests.blended().publish('requests blended rate')`,
						Rules: []*detector.Rule{
							{
								Severity:      "Info",
								Notifications: []*notification.Notification{},
							},
						},
					}
					_ = json.NewEncoder(w).Encode(&det)
				}),
				"POST /v2/signalflow/_/getSignalFlowModel": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					program, _ := io.ReadAll(r.Body)
					_ = r.Body.Close()
					if string(program) != "regen" {
						graph := flow.ExecutionGraph{
							flow.ExecutionBlockImport{
								Typed:  "IMPORT",
								Module: "signalfx.detectors.autodetect.apm",
								Name:   "requests",
							},
							flow.ExecutionBlockStream{
								Typed: "DETECT",
								Start: flow.ExecutionBlockStreamMethod{
									FunctionName: "requests.blended",
								},
								Methods: []flow.ExecutionBlockStreamMethod{
									{
										FunctionName: "publish",
										Arguments: flow.ExecutionBlockArguments{
											"label": &flow.ExecutionBlockArgumentLiteral{
												Value: "requests blended rate",
											},
										},
									},
								},
							},
						}
						_ = json.NewEncoder(w).Encode(&graph)
						return
					}
					graph := flow.ExecutionGraph{
						flow.ExecutionBlockImport{
							Typed:  "IMPORT",
							Module: "signalfx.detectors.autodetect.apm",
							Name:   "requests",
						},
						flow.ExecutionBlockStream{
							Typed: "DETECT",
							Start: flow.ExecutionBlockStreamMethod{
								FunctionName: "requests.blended",
								Arguments: flow.ExecutionBlockArguments{
									"filter_": &flow.ExecutionBlockArgumentOperation{
										Type:      "binary_expression",
										Operation: "AND",
										LeftRaw:   json.RawMessage(`{"type": "filter", "field": {"value": "service"}, "value": {"value": "web"}}`),
										Left: &flow.ExecutionBlockArgumentFilter{
											Type: "filter",
											Field: flow.ExecutionBlockArgumentLiteral{
												Value: "service",
											},
											Value: &flow.ExecutionBlockArgumentLiteral{
												Value: "web",
											},
										},
										RightRaw: json.RawMessage(`{"type": "filter", "field": {"value": "environment"}, "value": {"value": "production"}}`),
										Right: &flow.ExecutionBlockArgumentFilter{
											Type: "filter",
											Field: flow.ExecutionBlockArgumentLiteral{
												Value: "environment",
											},
											Value: &flow.ExecutionBlockArgumentLiteral{
												Value: "production",
											},
										},
									},
									"guard": &flow.ExecutionBlockArgumentLiteral{Value: 0.81},
								},
							},
							Methods: []flow.ExecutionBlockStreamMethod{
								{
									FunctionName: "publish",
									Arguments: flow.ExecutionBlockArguments{
										"label": &flow.ExecutionBlockArgumentLiteral{
											Value: "requests blended rate",
										},
									},
								},
							},
						},
					}
					_ = json.NewEncoder(w).Encode(&graph)
				}),
				"GET /v2/signalflow/_/extractMetadata": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					content := metadata.Metadata{
						Arguments: []metadata.Argument{
							{
								Name:        "guard",
								Type:        "double",
								Label:       "Argument 1",
								Description: "This is argument 1",
							},
							{
								Name: "filter_",
								Type: "filter",
							},
						},
					}
					_ = json.NewEncoder(w).Encode(&content)
				}),
				"POST /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					req := &detector.CreateUpdateDetectorRequest{}
					_ = json.NewDecoder(r.Body).Decode(req)
					_ = r.Body.Close()

					expectProgram := "from signalfx.detectors.autodetect.apm import requests\n" +
						`requests.blended(filter_=filter('environment', 'production') and filter('service', 'web'), guard=0.81).publish(label='requests blended rate')` + "\n"

					if req.ProgramText != expectProgram {
						diff := cmp.Diff(expectProgram, req.ProgramText)
						http.Error(w, "unexpected program text:\n"+diff, http.StatusBadRequest)
						return
					}

					det := detector.Detector{
						Id:               "auto-detector-1",
						ParentDetectorId: req.ParentDetectorId,
						DetectorOrigin:   "AutoDetectCustomization",
						Name:             req.Name,
						Description:      req.Description,
						ProgramText:      "regen",
						Rules:            req.Rules,
						Tags:             req.Tags,
						Teams:            req.Teams,
					}
					_ = json.NewEncoder(w).Encode(&det)
				}),
				"GET /v2/detector/auto-detector-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					det := detector.Detector{
						Id:               "auto-detector-1",
						ParentDetectorId: "parent-detector",
						DetectorOrigin:   "AutoDetectCustomization",
						Name:             "Modified Example Detector",
						Description:      "This is an example of a modified auto detector resource.",
						ProgramText:      "regen",
						Rules:            []*detector.Rule{},
						Tags:             []string{"tag-01", "tag-02"},
						Teams:            []string{"team-01", "team-02"},
					}
					_ = json.NewEncoder(w).Encode(&det)
				}),
				"DELETE /v2/detector/auto-detector-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			steps: []resourcetest.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/auto_detect_create.tf"),
					Check: resourcetest.ComposeAggregateTestCheckFunc(
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "id", "auto-detector-1"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "name", "Modified Example Detector"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "description", "This is an example of a modified auto detector resource."),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "parent_id", "parent-detector"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "tags.#", "2"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "tags.0", "tag-01"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "tags.1", "tag-02"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "teams.#", "2"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "teams.0", "team-01"),
						resourcetest.TestCheckResourceAttr("signalfx_customized_auto_detector.example", "teams.1", "team-02"),
					),
					ExpectNonEmptyPlan: true,
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
