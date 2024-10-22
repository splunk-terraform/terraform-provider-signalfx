// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// Using generics for the function signature instead of the typical any parameter
// to help reduce repeated lines of code needing to cast to or from an empty interface.
type (
	// DecodeTerraformFunc is used to define the method signature used to convert
	// the terraform state data into the expected API type to be used.
	//
	// This is to be used when interfacing with other packages.
	DecodeTerraformFunc[T any] func(rd *schema.ResourceData) (*T, error)

	// EncodeTerraformFunc is used to write the API response data into
	// the terraform state.
	//
	// This is to be used when interfacing with other packages.
	EncodeTerraformFunc[T any] func(t *T, rd *schema.ResourceData) error
)

func NopDecodeTerraform[T any](*schema.ResourceData) (*T, error) {
	return nil, nil
}

func NopEncodeTerraform[T any](*T, *schema.ResourceData) error {
	return nil
}
