// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionBlockArguments(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  string
		expect ExecutionBlockArguments
	}{
		{
			name:  "literal string argument",
			input: `{"arg1": {"type": "string", "value": "test"}}`,
			expect: ExecutionBlockArguments{
				"arg1": &ExecutionBlockArgumentLiteral{
					Type:  "string",
					Value: "test",
				},
			},
		},
		{
			name:  "literal int argument",
			input: `{"arg1": {"type": "int", "value": 10}}`,
			expect: ExecutionBlockArguments{
				"arg1": &ExecutionBlockArgumentLiteral{
					Type:  "int",
					Value: float64(10),
				},
			},
		},
		{
			name:  "literal bool argument",
			input: `{"arg1": {"type": "bool", "value": true}}`,
			expect: ExecutionBlockArguments{
				"arg1": &ExecutionBlockArgumentLiteral{
					Type:  "bool",
					Value: true,
				},
			},
		},
		{
			name:  "multiple arguments",
			input: `{"arg1": {"type": "string", "value": "test"}, "arg2": {"type": "int", "value": 10}}`,
			expect: ExecutionBlockArguments{
				"arg1": &ExecutionBlockArgumentLiteral{
					Type:  "string",
					Value: "test",
				},
				"arg2": &ExecutionBlockArgumentLiteral{
					Type:  "int",
					Value: float64(10),
				},
			},
		},
		{
			name:  "operation argument",
			input: `{"arg1": {"type": "binary_expression", "originalText": "10 > 5", "position": 1, "left": {"type": "int", "value": 10}, "op": ">", "right": {"type": "int", "value": 5}}}`,
			expect: ExecutionBlockArguments{
				"arg1": &ExecutionBlockArgumentOperation{
					Type:         "binary_expression",
					OriginalText: "10 > 5",
					Position:     1,
					LeftRaw:      json.RawMessage(`{"type": "int", "value": 10}`),
					Left: &ExecutionBlockArgumentLiteral{
						Type:  "int",
						Value: float64(10),
					},
					Operation: ">",
					RightRaw:  json.RawMessage(`{"type": "int", "value": 5}`),
					Right: &ExecutionBlockArgumentLiteral{
						Type:  "int",
						Value: float64(5),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := make(ExecutionBlockArguments)
			require.NoError(t, json.NewDecoder(strings.NewReader(tc.input)).Decode(&actual))
			assert.Equal(t, tc.expect, actual, "Must match the expected values")
		})
	}
}

func TestExecutionBlockArgumentExpressions(t *testing.T) {
	t.Parallel()

	stringLiteral := &ExecutionBlockArgumentLiteral{Value: "value"}
	numberLiteral := &ExecutionBlockArgumentLiteral{Value: 42}
	assert.Equal(t, "'value'", stringLiteral.Expression())
	assert.Equal(t, "42", numberLiteral.Expression())

	filter := &ExecutionBlockArgumentFilter{
		Field:  ExecutionBlockArgumentLiteral{Value: "service.name"},
		Value:  &ExecutionBlockArgumentLiteral{Value: "api"},
		Values: []ExecutionBlockArgumentLiteral{{Value: "worker"}},
	}
	assert.Equal(t, `filter("service.name",'api' , 'worker')`, filter.Expression())

	operation := &ExecutionBlockArgumentOperation{Left: numberLiteral, Operation: ">", Right: &ExecutionBlockArgumentLiteral{Value: 10}}
	assert.Equal(t, "(42 > 10)", operation.Expression())

	for _, value := range []ExecutionBlockArgumentValue{stringLiteral, filter, operation} {
		value.isArgumentValue()
	}
}
