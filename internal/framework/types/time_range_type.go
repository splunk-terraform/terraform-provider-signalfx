// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtypes

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type TimeRangeType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = (*TimeRangeType)(nil)

func (t TimeRangeType) String() string {
	return "fwtypes.TimeRangeType"
}

func (t TimeRangeType) ValueType(ctx context.Context) attr.Value {
	return TimeRange{}
}

func (t TimeRangeType) Equal(o attr.Type) bool {
	other, ok := o.(TimeRangeType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t TimeRangeType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return TimeRange{
		StringValue: in,
	}, nil
}

func (t TimeRangeType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	strVal, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("expected basetypes.StringValue, got %T", attrValue)
	}

	valuable, diags := t.ValueFromString(ctx, strVal)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return valuable, nil
}
