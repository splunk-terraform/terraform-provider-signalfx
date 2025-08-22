// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0
package fwshared

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/stretchr/testify/assert"
)

func TestResourceIDAttribute(t *testing.T) {
	type args struct {
		opts []func(*schema.StringAttribute)
	}
	tests := []struct {
		name           string
		args           args
		wantComputed   bool
		wantDesc       string
		wantPlanMods   int
		wantCustomDesc string
	}{
		{
			name:         "default options",
			args:         args{opts: nil},
			wantComputed: true,
			wantDesc:     "The unique identifier for the resource.",
			wantPlanMods: 1,
		},
		{
			name: "custom description",
			args: args{opts: []func(*schema.StringAttribute){
				func(sa *schema.StringAttribute) {
					sa.Description = "Custom description"
				},
			}},
			wantComputed:   true,
			wantDesc:       "Custom description",
			wantPlanMods:   1,
			wantCustomDesc: "Custom description",
		},
		{
			name: "add extra plan modifier",
			args: args{opts: []func(*schema.StringAttribute){
				func(sa *schema.StringAttribute) {
					sa.PlanModifiers = append(sa.PlanModifiers, stringplanmodifier.RequiresReplace())
				},
			}},
			wantComputed: true,
			wantDesc:     "The unique identifier for the resource.",
			wantPlanMods: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResourceIDAttribute(tt.args.opts...)
			assert.Equal(t, tt.wantComputed, got.Computed)
			assert.Equal(t, tt.wantDesc, got.Description)
			assert.Len(t, got.PlanModifiers, tt.wantPlanMods)
		})
	}
}
