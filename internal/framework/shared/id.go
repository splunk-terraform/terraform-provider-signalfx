// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func ResourceIDAttribute(opts ...func(*schema.StringAttribute)) schema.StringAttribute {
	id := schema.StringAttribute{
		Computed:    true,
		Description: "The unique identifier for the resource.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	for _, opt := range opts {
		opt(&id)
	}
	return id
}
