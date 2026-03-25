// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func AttributeTypeMap[T interface{ GetType() attr.Type }](t map[string]T) map[string]attr.Type {
	types := make(map[string]attr.Type)
	for key, value := range t {
		types[key] = value.GetType()
	}
	return types
}

func StringValueOrEmpty(str string) types.String {
	if str == "" {
		return types.StringNull()
	}
	return types.StringValue(str)
}
