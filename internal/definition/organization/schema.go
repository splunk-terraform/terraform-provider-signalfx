// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"emails": {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Email,
			},
			Required:    true,
			Description: "A list of email address that can be matched against existing users of the organization.",
		},
		"users": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Provides a list of user IDs that match the emails provided. The user IDs are returned in the order of the provided emails.",
		},
	}
}
