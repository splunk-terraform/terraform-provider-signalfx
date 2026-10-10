// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

// integrationModel contains state shared by notification integrations with
// identical API and Terraform semantics.
type integrationModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func integrationAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": fwshared.ResourceIDAttribute(),
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Human-readable name of the integration.",
		},
		"enabled": schema.BoolAttribute{
			Required:    true,
			Description: "Whether the integration is enabled.",
		},
	}
}

func (model *integrationModel) update(name string, enabled bool) {
	model.Name = types.StringValue(name)
	model.Enabled = types.BoolValue(enabled)
}

func (model *integrationModel) updateWithID(id, name string, enabled bool) {
	model.ID = types.StringValue(id)
	model.update(name, enabled)
}
