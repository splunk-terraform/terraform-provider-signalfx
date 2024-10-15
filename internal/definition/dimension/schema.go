// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dimension

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"query": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "",
		},
		"order_by": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      1000,
			Description:  "This allows you to define how many dimensions are returned as the values output.",
			ValidateFunc: validation.IntBetween(0, 10_000),
		},
		"values": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of all the match dimension values that the provided query, ordered by order_by field",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}
}
