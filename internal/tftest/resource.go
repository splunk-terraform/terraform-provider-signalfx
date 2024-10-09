// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

// ResourceOperationTestCase allows for easy means of mocking operations with a resource.
// To simplify working with terraform, it uses Encoder and Decoder to covert the api type T
// into teraform expected format.
type ResourceOperationTestCase[T any] struct {
	// Name is used to set the test name
	Name string
	// Meta is a shortcut to providing the
	// post configured provider details that would be passed around.
	// The value TB is passed in to allow for cleanup of any created resources
	Meta func(tb testing.TB) any
	// Definition is used to isolate the resource being tested
	Resource *schema.Resource
	// Encoder is used to simplify generating the [*schema.ResourceData]
	Encoder tfext.EncodeTerraformFunc[T]
	// Decoder is used to parse the final result of the operation
	Decoder tfext.DecodeTerraformFunc[T]
	// Input is the initial "state" value that would
	// either be provided by the configuration or state file.
	Input *T
	// Expected is the final value that should have set
	// once the operation completed
	Expect *T
	// Issues is any expected issues once the operation is complete
	Issues diag.Diagnostics
}

func (tc ResourceOperationTestCase[T]) TestCreate(t *testing.T) {
	var operation schema.CreateContextFunc = func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		return diag.Errorf("no create operation defined")
	}

	//nolint // This method is deprecated but is still used by some resources
	if !reflect.ValueOf(tc.Resource.Create).IsNil() {
		operation = func(_ context.Context, rd *schema.ResourceData, meta any) diag.Diagnostics {
			//nolint // This method is deprecated but is still used by some resources
			return diag.FromErr(tc.Resource.Create(rd, meta))
		}
	}

	if !reflect.ValueOf(tc.Resource.CreateContext).IsNil() {
		operation = tc.Resource.CreateContext
	}

	if !reflect.ValueOf(tc.Resource.CreateWithoutTimeout).IsNil() {
		operation = tc.Resource.CreateWithoutTimeout
	}

	t.Run(tc.Name, func(t *testing.T) {
		t.Parallel()

		tc.testOperation(t, tc.Resource, operation)
	})
}

func (tc ResourceOperationTestCase[T]) TestRead(t *testing.T) {
	var operation schema.ReadContextFunc = func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		return diag.Errorf("no read operation defined")
	}

	//nolint // This method is deprecated but is still used by some resources
	if !reflect.ValueOf(tc.Resource.Read).IsNil() {
		operation = func(_ context.Context, rd *schema.ResourceData, meta any) diag.Diagnostics {
			//nolint // This method is deprecated but is still used by some resources
			return diag.FromErr(tc.Resource.Read(rd, meta))
		}
	}

	if !reflect.ValueOf(tc.Resource.ReadContext).IsNil() {
		operation = tc.Resource.ReadContext
	}

	if !reflect.ValueOf(tc.Resource.ReadWithoutTimeout).IsNil() {
		operation = tc.Resource.ReadWithoutTimeout
	}

	t.Run(tc.Name, func(t *testing.T) {
		tc.testOperation(t, tc.Resource, operation)
	})
}

func (tc ResourceOperationTestCase[T]) TestUpdate(t *testing.T) {
	var operation schema.UpdateContextFunc = func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		return diag.Errorf("no update operation defined")
	}

	//nolint // This method is deprecated but is still used by some resources
	if !reflect.ValueOf(tc.Resource.Update).IsNil() {
		operation = func(_ context.Context, rd *schema.ResourceData, meta any) diag.Diagnostics {
			//nolint // This method is deprecated but is still used by some resources
			return diag.FromErr(tc.Resource.Update(rd, meta))
		}
	}

	if !reflect.ValueOf(tc.Resource.UpdateContext).IsNil() {
		operation = tc.Resource.UpdateContext
	}

	if !reflect.ValueOf(tc.Resource.UpdateWithoutTimeout).IsNil() {
		operation = tc.Resource.UpdateWithoutTimeout
	}

	t.Run(tc.Name, func(t *testing.T) {
		tc.testOperation(t, tc.Resource, operation)
	})
}

func (tc ResourceOperationTestCase[T]) TestDelete(t *testing.T) {
	var operation schema.DeleteContextFunc = func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		return diag.Errorf("no delete operation defined")
	}

	//nolint // This method is deprecated but is still used by some resources
	if !reflect.ValueOf(tc.Resource.Delete).IsNil() {
		operation = func(_ context.Context, rd *schema.ResourceData, meta any) diag.Diagnostics {
			//nolint // This method is deprecated but is still used by some resources
			return diag.FromErr(tc.Resource.Delete(rd, meta))
		}
	}

	if !reflect.ValueOf(tc.Resource.DeleteContext).IsNil() {
		operation = tc.Resource.DeleteContext
	}

	if !reflect.ValueOf(tc.Resource.DeleteWithoutTimeout).IsNil() {
		operation = tc.Resource.DeleteWithoutTimeout
	}

	t.Run(tc.Name, func(t *testing.T) {
		tc.testOperation(t, tc.Resource, operation)
	})
}

func (tc ResourceOperationTestCase[T]) testOperation(
	t *testing.T,
	resource *schema.Resource,
	op func(context.Context, *schema.ResourceData, any) diag.Diagnostics,
) {
	rd := resource.TestResourceData()
	require.NoError(t, tc.Encoder(tc.Input, rd), "Must not error encoding input value into resource data")

	actual := op(context.Background(), rd, tc.Meta(t))
	assert.Equal(t, tc.Issues, actual, "Must match the expected issues defined")

	if len(tc.Issues) == 0 {
		data, err := tc.Decoder(rd)
		assert.Equal(t, tc.Expect, data, "Must match the expected value")
		require.NoError(t, err, "Must not error parsing the data")
	}
}
