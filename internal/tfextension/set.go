// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// NewSchemaSet converts any array of values into a set with the provided hash function.
func NewSchemaSet[S ~[]E, E any](hash schema.SchemaSetFunc, s S) *schema.Set {
	items := make([]any, len(s))
	for i, v := range s {
		items[i] = v
	}
	return schema.NewSet(hash, items)
}
