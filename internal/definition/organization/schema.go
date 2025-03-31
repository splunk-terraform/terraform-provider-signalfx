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
			Required: true,
		},
		"users": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}
}
