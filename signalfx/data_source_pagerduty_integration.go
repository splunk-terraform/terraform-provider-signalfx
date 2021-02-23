package signalfx

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourcePagerDutyIntegration() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyIntegrationRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourcePagerDutyIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	int, err := config.Client.GetPagerDutyIntegrationByName(context.TODO(), d.Get("name").(string))
	if err != nil {
		return err
	}

	if int == nil {
		d.SetId("")

		return nil
	}

	d.SetId(int.Id)
	return pagerDutyIntegrationAPIToTF(d, int)
}
