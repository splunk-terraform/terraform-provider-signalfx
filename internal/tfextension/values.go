// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// Values is a unified resource data type
// to allow for high reusable methods in different scenarios.
//
// For method information, please refer to the defined types
// below and check their API interface.
type Values interface {
	Id() string
	Get(string) any
	GetOk(string) (any, bool)
	HasChanges(values ...string) bool
}

var (
	_ Values = (*schema.ResourceData)(nil)
	_ Values = (*schema.ResourceDiff)(nil)
)
