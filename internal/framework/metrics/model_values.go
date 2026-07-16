// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import "github.com/hashicorp/terraform-plugin-framework/types"

func optionalStringValue(current types.String, value *string) types.String {
	if value != nil {
		return types.StringValue(*value)
	}
	if current.IsUnknown() {
		return types.StringNull()
	}
	return current
}

func optionalInt64Value(current types.Int64, value *int64) types.Int64 {
	if value != nil {
		return types.Int64Value(*value)
	}
	if current.IsUnknown() {
		return types.Int64Null()
	}
	return current
}
