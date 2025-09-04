// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
)

func TestWalkedValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		value  WalkedValue
		expect string
	}{
		{
			name:   "zero value",
			value:  WalkedValue{},
			expect: "path: value:<nil> error:<nil>",
		},
		{
			name:   "has value set",
			value:  NewWalkedValue(path.Empty(), "hehe"),
			expect: "path: value:hehe error:<nil>",
		},
		{
			name:   "has error set",
			value:  NewWalkedValueErrorWithPath(path.Root("root"), "this has gone on for %d long", 2),
			expect: "path:root value:<nil> error:this has gone on for 2 long",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, tc.value.String(), "WalkedValue String representation should match expected")
		})
	}
}

func TestWalkStruct(t *testing.T) {
	t.Parallel()

	t.Run("non trivial type", func(t *testing.T) {
		t.Parallel()

		type inlined struct {
			Value types.String `tfsdk:"value"`
			Count types.Number `tfsdk:"count"`
		}
		type ComplexType struct {
			List   []*inlined   `tfsdk:"list"`
			Nested *inlined     `tfsdk:"nested"`
			Scaler types.Number `tfsdk:"scalar"`
		}
		var (
			errs   error
			expect = map[string]struct{}{
				"list[0]":       {},
				"list[0].value": {},
				"list[0].count": {},
				"nested":        {},
				"nested.value":  {},
				"nested.count":  {},
				"scalar":        {},
			}
		)
		for wv := range WalkStruct(&ComplexType{Nested: &inlined{}}) {
			p := wv.Path().String()
			_, ok := expect[p]
			assert.True(t, ok, "Must only see expected fields, got unexpected %q", p)
			delete(expect, wv.Path().String())
			errs = multierr.Append(errs, wv.Err())
		}
		assert.Empty(t, expect, "Must have seen all expected fields")
		assert.NoError(t, errs, "Must not have any errors")
	})
}
