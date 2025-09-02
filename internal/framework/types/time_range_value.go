// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtypes

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type TimeRange struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuableWithSemanticEquals = (*TimeRange)(nil)
	_ xattr.ValidateableAttribute                = (*TimeRange)(nil)
	_ function.ValidateableParameter             = (*TimeRange)(nil)
)

func (tr TimeRange) Type(_ context.Context) attr.Type {
	return TimeRangeType{}
}

func (tr TimeRange) Equal(o attr.Value) bool {
	other, ok := o.(TimeRange)

	if !ok {
		return false
	}

	return tr.StringValue.Equal(other.StringValue)
}

func (tr TimeRange) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if tr.IsUnknown() || tr.IsNull() {
		return
	}

	if _, err := tr.ValueTimeRange(); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Time Range", err.Error())
	}
}

func (tr TimeRange) ValidateParameter(ctx context.Context, req function.ValidateParameterRequest, resp *function.ValidateParameterResponse) {
	if tr.IsUnknown() || tr.IsNull() {
		return
	}

	if _, err := tr.ValueTimeRange(); err != nil {
		resp.Error = function.NewArgumentFuncError(req.Position, err.Error())
	}
}

func (tr TimeRange) ValueTimeRange() (time.Duration, error) {
	var (
		units = map[rune]time.Duration{
			's': 1 * time.Second,
			'm': 1 * time.Minute,
			'h': 1 * time.Hour,
			'd': 24 * time.Hour,
			'w': 7 * 24 * time.Hour,
		}
		raw     = tr.StringValue.ValueString()
		partial time.Duration
		total   time.Duration
	)

	for _, r := range strings.TrimPrefix(raw, "-+") {
		_, isUnit := units[r]
		switch {
		case unicode.IsDigit(r):
			partial = partial*10 + time.Duration(r-'0')
		case isUnit:
			if partial == 0 {
				return 0, fmt.Errorf("invalid timerange: %q", raw)
			}
			total, partial = (total + partial*units[r]), 0
		default:
			return 0, fmt.Errorf("invalid timerange: unexpected character: %q", r)
		}
	}

	if partial != 0 {
		return 0, fmt.Errorf("invalid timerange:expected unit: %q", raw)
	}

	if strings.HasPrefix(raw, "-") {
		total *= -1
	}

	return total, nil
}

func (tr TimeRange) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	nv, ok := newValuable.(TimeRange)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An expected value type was received while comparing semantic values",
		)
	}

	old, _ := tr.ValueTimeRange()
	new, _ := nv.ValueTimeRange()

	return old == new, diags
}
