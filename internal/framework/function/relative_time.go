// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalfunction

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	fwtypes "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/types"
)

type TimeRangeParser struct{}

var _ function.Function = (*TimeRangeParser)(nil)

func NewTimeRangeParser() function.Function {
	return &TimeRangeParser{}
}

func (TimeRangeParser) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "parse_time_range"
}

func (TimeRangeParser) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Convert relative time value into milliseconds",
		Description: "In order to make some of the time values easier to work with, this will parse the relative time signature (ie: `-1h30m`) and return the number of milliseconds that it represents.",
		Parameters: []function.Parameter{
			function.StringParameter{
				AllowNullValue: false,
				CustomType:     fwtypes.TimeRangeType{},
				Name:           "time_range",
				Description:    "Used to parse the given relative time string into a the amount of milliseconds.",
			},
		},
		Return: function.Int64Return{},
	}
}

func (TimeRangeParser) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var relative string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &relative))

	tr := fwtypes.TimeRange{
		StringValue: basetypes.NewStringValue(relative),
	}

	if val, err := tr.ParseDuration(); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
	} else {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, val.Milliseconds()))
	}

}
