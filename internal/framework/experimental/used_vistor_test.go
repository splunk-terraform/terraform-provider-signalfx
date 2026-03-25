// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
)

func TestNewUsedVisitor(t *testing.T) {
	t.Parallel()

	actual := NewUsedVisitor()

	require.NotNil(t, actual)
	assert.NotNil(t, actual.mem)
	assert.Empty(t, actual.mem)

	var inputs [][3]string
	for input := range actual.Inputs() {
		inputs = append(inputs, input)
	}
	assert.Empty(t, inputs)
}

func TestUsedVisitorVisit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		blocks      flow.ExecutionGraph
		wantInputs  [][3]string
		wantArgs    map[string]metadata.Argument
		wantFilters map[string][]string
	}{
		{
			name: "import without alias stores module by name",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
			wantArgs:    map[string]metadata.Argument{},
			wantFilters: map[string][]string{},
		},
		{
			name: "import with alias stores module by alias",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Alias:  "apm",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
			wantArgs:    map[string]metadata.Argument{},
			wantFilters: map[string][]string{},
		},
		{
			name: "non detect stream is ignored",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
				flow.ExecutionBlockStream{
					Typed: "PLOT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
					},
				},
			},
			wantArgs:    map[string]metadata.Argument{},
			wantFilters: map[string][]string{},
		},
		{
			name: "detect stream with known import adds input",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Alias:  "apm",
					Module: "signalfx.detectors.autodetect.apm",
				},
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "apm.blended",
					},
				},
			},
			wantInputs: [][3]string{
				{"signalfx.detectors.autodetect.apm", "apm", "blended"},
			},
			wantArgs:    map[string]metadata.Argument{},
			wantFilters: map[string][]string{},
		},
		{
			name: "detect stream with unknown import does not add input",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "unknown.blended",
					},
				},
			},
			wantInputs:  nil,
			wantArgs:    map[string]metadata.Argument{},
			wantFilters: map[string][]string{},
		},
		{
			name: "Detect stream with arguments set",
			blocks: []flow.ExecutionBlock{
				flow.ExecutionBlockStream{
					Typed: "DETECT",
					Start: flow.ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						Arguments: flow.ExecutionBlockArguments{
							"limit": &flow.ExecutionBlockArgumentLiteral{
								Type: "int", Value: 10,
							},
							"filter": &flow.ExecutionBlockArgumentOperation{
								Operation: "AND",
								Left: &flow.ExecutionBlockArgumentFilter{
									Field: flow.ExecutionBlockArgumentLiteral{
										Value: "service.name",
									},
									Values: []flow.ExecutionBlockArgumentLiteral{
										{Value: "checkout"},
										{Value: "payment"},
										{Value: "cart"},
									},
								},
								Right: &flow.ExecutionBlockArgumentFilter{
									Field: flow.ExecutionBlockArgumentLiteral{
										Value: "env",
									},
									Value: &flow.ExecutionBlockArgumentLiteral{
										Value: "production",
									},
								},
							},
						},
					},
				},
			},
			wantInputs: nil,
			wantArgs: map[string]metadata.Argument{
				"limit": {Name: "limit", Type: "int", DefaultValue: 10},
			},
			wantFilters: map[string][]string{
				"service.name": {"checkout", "payment", "cart"},
				"env":          {"production"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			visitor := NewUsedVisitor()

			require.NoError(t, tc.blocks.Visit(visitor))

			assert.Equal(t, tc.wantInputs, slices.Collect(visitor.Inputs()), "Must match the expected inputs")
			assert.Equal(t, tc.wantArgs, visitor.Arguments(), "Must match the expected arguments")
			assert.Equal(t, tc.wantFilters, visitor.Filters(), "Must match the expected filters")
		})
	}
}
