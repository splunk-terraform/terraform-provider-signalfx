package signalfx

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func dataSourceAwsServices() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsServicesRead,
		Schema: map[string]*schema.Schema{
			"services": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsServicesRead(d *schema.ResourceData, meta interface{}) error {
	services := make([]map[string]interface{}, len(integration.AWSServiceNames))
	i := 0
	for k := range integration.AWSServiceNames {
		services[i] = map[string]interface{}{
			"name": k,
		}
		i++
	}
	d.SetId("AWS")
	d.Set("services", services)
	return nil
}
