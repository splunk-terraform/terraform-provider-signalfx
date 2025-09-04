// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
)

func TestWalkResourceSchema(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  map[string]rschema.Attribute
		expect map[string]rschema.Attribute
	}{
		{
			name:   "basic",
			input:  map[string]rschema.Attribute{"foo": rschema.StringAttribute{}},
			expect: map[string]rschema.Attribute{"foo": rschema.StringAttribute{}},
		},
		{
			name: "nested value",
			input: map[string]rschema.Attribute{
				"feature": rschema.SingleNestedAttribute{
					Attributes: map[string]rschema.Attribute{
						"enabled": rschema.BoolAttribute{},
					},
				},
				"items": rschema.ListNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"item": rschema.StringAttribute{},
						},
					},
				},
				"sets": rschema.SetNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"item": rschema.StringAttribute{},
						},
					},
				},
				"maps": rschema.MapNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"key":   rschema.StringAttribute{},
							"value": rschema.StringAttribute{},
						},
					},
				},
			},
			expect: map[string]rschema.Attribute{
				"feature": rschema.SingleNestedAttribute{
					Attributes: map[string]rschema.Attribute{
						"enabled": rschema.BoolAttribute{},
					},
				},
				"feature.enabled": rschema.BoolAttribute{},
				"items": rschema.ListNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"item": rschema.StringAttribute{},
						},
					},
				},
				"items[0].item": rschema.StringAttribute{},
				"sets": rschema.SetNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"item": rschema.StringAttribute{},
						},
					},
				},
				"sets.item": rschema.StringAttribute{},
				"maps": rschema.MapNestedAttribute{
					NestedObject: rschema.NestedAttributeObject{
						Attributes: map[string]rschema.Attribute{
							"key":   rschema.StringAttribute{},
							"value": rschema.StringAttribute{},
						},
					},
				},
				"[\"maps\"].key":   rschema.StringAttribute{},
				"[\"maps\"].value": rschema.StringAttribute{},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := make(map[string]rschema.Attribute)
			for p, v := range WalkResourceSchema(tc.input) {
				actual[p.String()] = v
			}
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
