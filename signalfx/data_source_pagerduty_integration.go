// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePagerDutyIntegration() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyIntegrationRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "This is the configured name of the PagerDuty integration.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the integration is currently enabled.",
			},
		},
		Description: "Use this data source to fetch the PagerDuty integration details.",
	}
}

func dataSourcePagerDutyIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	in, err := config.Client.GetPagerDutyIntegrationByName(context.TODO(), d.Get("name").(string))
	if err != nil {
		return err
	}

	if in == nil {
		d.SetId("")
		return nil
	}

	d.SetId(in.Id)
	return pagerDutyIntegrationAPIToTF(d, in)
}
