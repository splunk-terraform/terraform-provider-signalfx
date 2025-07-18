// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalfunction

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestTimeRangeParser_Metadata(t *testing.T) {
	t.Parallel()

	resp := &function.MetadataResponse{}
	NewTimeRangeParser().Metadata(t.Context(), function.MetadataRequest{}, resp)

	assert.Equal(t, "parse_time_range", resp.Name, "Function name must match")
}

func TestTimeRangeParser_Definition(t *testing.T) {
	t.Parallel()

	resp := &function.DefinitionResponse{}
	NewTimeRangeParser().Definition(t.Context(), function.DefinitionRequest{}, resp)

	assert.Equal(t, "Convert relative time value into milliseconds", resp.Definition.Summary, "Summary must match")
	assert.Equal(t, "In order to make some of the time values easier to work with, this will parse the relative time signature (ie: `-1h30m`) and return the number of milliseconds that it represents.", resp.Definition.Description, "Description must match")
	assert.Len(t, resp.Definition.Parameters, 1, "Must have one parameter")
	assert.Equal(t,
		function.StringParameter{
			AllowNullValue: false,
			Name:           "time_range",
			Description:    "Used to parse the given relative time string into a the amount of milliseconds.",
		},
		resp.Definition.Parameters[0],
		"Parameter name must match",
	)
}

func TestTimeRangeParser_Run(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		arg    function.RunRequest
		expect *function.RunResponse
	}{
		{
			name: "invalid timerange provided",
			arg: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("0"),
				}),
			},
			expect: &function.RunResponse{
				Result: function.NewResultData(types.Int64Unknown()),
				Error:  function.NewFuncError(`invalid timerange "0": no negative prefix`),
			},
		},
		{
			name: "missing sign in timerange",
			arg: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("1h"),
				}),
			},
			expect: &function.RunResponse{
				Result: function.NewResultData(types.Int64Unknown()),
				Error:  function.NewFuncError(`invalid timerange "1h": no negative prefix`),
			},
		},
		{
			name: "valid timerange",
			arg: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("-1h"),
				}),
			},
			expect: &function.RunResponse{
				Result: function.NewResultData(types.Int64Value(3600000)),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := &function.RunResponse{
				Error:  nil,
				Result: function.NewResultData(types.Int64Unknown()),
			}

			NewTimeRangeParser().Run(t.Context(), tt.arg, actual)
			assert.Equal(t, tt.expect, actual, "Must match the expected results")
		})
	}
}
