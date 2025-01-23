// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

// ResourceDataAccess is an interface designed to abstract access to resource data.
// It is implemented by both *schema.ResourceDiff and *schema.ResourceData types,
// allowing functions to operate on them interchangeably.
//
// This interface provides methods to retrieve resource properties based on a key.
type ResourceDataAccess interface {
	Get(key string) interface{}
	GetOk(key string) (interface{}, bool)
}
