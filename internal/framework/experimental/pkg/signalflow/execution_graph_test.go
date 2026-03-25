// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExectionGraphUnmarshalJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		jsonData string
		expect   []ExecutionBlock
		errVal   string
	}{
		{
			name:     "Module definition",
			jsonData: `[{"alias": null,"module": "signalfx.detectors.autodetect.apm","name": "requests","type": "IMPORT"}]`,
			expect: []ExecutionBlock{
				ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
			},
		},
		{
			name: "Module with function definition",
			jsonData: `[
			{"alias": null,"module": "signalfx.detectors.autodetect.apm","name": "requests","type": "IMPORT"},
			{
			"expressionText": null,
			"start": {
				"functionName": "requests.blended",
				"originalText": "requests.blended().publish('Request Rate Dropped')",
				"position": 56,
				"type": "user_function"
			},
			"streamMethods": [
				{
					"args": {
						"label": {
							"originalText": "'Request Rate Dropped'",
							"position": 83,
							"type": "string",
							"value": "Request Rate Dropped"
						}
					},
					"functionName": "publish",
					"originalText": "requests.blended().publish('Request Rate Dropped')",
					"position": 56,
					"type": "stream_method"
				}
			],
			"type": "DETECT",
			"uniqueKey": 0
		}]`,
			expect: []ExecutionBlock{
				ExecutionBlockImport{
					Typed:  "IMPORT",
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
				ExecutionBlockStream{
					Typed: "DETECT",
					Start: ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended().publish('Request Rate Dropped')",
						Position:     56,
						Type:         "user_function",
					},
					Methods: []ExecutionBlockStreamMethod{
						{
							FunctionName: "publish",
							OriginalText: "requests.blended().publish('Request Rate Dropped')",
							Position:     56,
							Type:         "stream_method",
							Arguments: ExecutionBlockArguments{
								"label": &ExecutionBlockArgumentLiteral{
									OriginalText: "'Request Rate Dropped'",
									Position:     83,
									Type:         "string",
									Value:        "Request Rate Dropped",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := NewExecutionGraphFromJSON(strings.NewReader(tc.jsonData))
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NoError(t, err, "Must not return an error")
			}

			require.Len(t, actual, len(tc.expect), "Must match the expected length")
			for i, expect := range tc.expect {
				assert.Equal(t, expect, actual[i], "Must match the expected block type at index %d", i)
			}
		})
	}
}

func TestExecutionGraphVisit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		visitor Visitor
		errVal  string
	}{
		{
			name: "blank visitor",
			visitor: VisitorFunc(func(block ExecutionBlock) error {
				return nil
			}),
			errVal: "",
		},
		{
			name: "erroring visitor",
			visitor: VisitorFunc(func(block ExecutionBlock) error {
				return assert.AnError
			}),
			errVal: "assert.AnError general error for testing",
		},
		{
			name: "Check types",
			visitor: VisitorFunc(func(block ExecutionBlock) error {
				switch b := block.(type) {
				case ExecutionBlockImport:
					if b.Name != "requests" {
						return assert.AnError
					}
					if b.Module != "signalfx.detectors.autodetect.apm" {
						return assert.AnError
					}
				case ExecutionBlockStream:
					if b.Type() != "DETECT" {
						return assert.AnError
					}
					if b.Start.FunctionName != "requests.blended" {
						return assert.AnError
					}
					if b.Start.OriginalText != "requests.blended().publish('Request Rate Dropped')" {
						return assert.AnError
					}
					if b.Start.Position != 56 {
						return assert.AnError
					}
					if b.Start.Type != "user_function" {
						return assert.AnError
					}
				}
				return nil
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			graph := ExecutionGraph{
				ExecutionBlockImport{
					Name:   "requests",
					Module: "signalfx.detectors.autodetect.apm",
				},
				ExecutionBlockStream{
					Typed: "DETECT",
					Start: ExecutionBlockStreamMethod{
						FunctionName: "requests.blended",
						OriginalText: "requests.blended().publish('Request Rate Dropped')",
						Position:     56,
						Type:         "user_function",
					},
				},
			}

			if err := graph.Visit(tc.visitor); tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NoError(t, err, "Must not return an error")
			}
		})
	}
}
