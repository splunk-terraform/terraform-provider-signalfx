// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
)

func TestNewProgramBuilderVisitor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		graph   flow.ExecutionGraph
		inputs  map[string]any
		filters map[string][]string
		expect  string
	}{
		{
			name:   "empty graph produces empty program",
			graph:  flow.ExecutionGraph{},
			expect: "",
		},
		{
			name: "import block produces import statement",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
			expect: "import signalfx.detectors.autodetect.apm\n",
		},
		{
			name: "import block produces import statement",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
			expect: "from signalfx.detectors.autodetect.apm import requests\n",
		},
		{
			name: "import block with alias produces import statement with alias",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Alias:  "apm",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
			expect: "from signalfx.detectors.autodetect.apm import requests as apm\n",
		},
		{
			name: "detect block produces function call with filters",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended()",
					},
				},
			},
			filters: map[string][]string{
				"env": {"prod", "staging"},
			},
			expect: "requests.blended(_filter=filter('env', 'prod', 'staging'))\n",
		},
		{
			name: "detect block with multiple filters produces function call with filters combined by and",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended()",
					},
				},
			},
			filters: map[string][]string{
				"env":     {"prod", "staging"},
				"service": {"checkout"},
			},
			expect: "requests.blended(_filter=filter('env', 'prod', 'staging') and filter('service', 'checkout'))\n",
		},
		{
			name: "detect function with inputs",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended()",
					},
				},
			},
			inputs: map[string]any{
				"target": 80.0,
			},
			expect: "requests.blended(target=80)\n",
		},
		{
			name: "detect function with inputs",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended()",
					},
				},
			},
			inputs: map[string]any{
				"target":          80.0,
				"feature_enabled": true,
			},
			expect: "requests.blended(feature_enabled=True, target=80)\n",
		},
		{
			name: "detect function with methods with arguments",
			graph: flow.ExecutionGraph{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended()",
					},
					Methods: []flow.ExecutionBlockStreamMethod{
						{
							FunctionName: "method1",
							OriginalText: "method1(arg1=10)",
							Arguments: map[string]flow.ExecutionBlockArgumentValue{
								"arg1": &flow.ExecutionBlockArgumentLiteral{
									Type:         "int",
									Value:        10,
									OriginalText: "10",
								},
							},
						},
						{
							FunctionName: "method2",
							OriginalText: "method2(arg2='value')",
							Arguments: map[string]flow.ExecutionBlockArgumentValue{
								"arg2": &flow.ExecutionBlockArgumentLiteral{
									Type:         "string",
									Value:        "value",
									OriginalText: "'value'",
								},
							},
						},
					},
				},
			},
			expect: "requests.blended().method1(arg1=10).method2(arg2='value')\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vb := NewProgramBuilderVisitor(
				func(v *ProgramBuilderVisitor) {
					maps.Copy(v.inputs, tc.inputs)
					maps.Copy(v.filters, tc.filters)
				},
			).WithFilterKey("_filter")

			err := tc.graph.Visit(vb)
			require.NoError(t, err, "Must not error when processing graph")

			assert.Equal(t, tc.expect, vb.BuildProgramText(), "Must match the expected graph")
		})
	}
}
