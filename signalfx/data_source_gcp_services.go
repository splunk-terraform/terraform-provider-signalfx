package signalfx

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func dataSourceGcpServices() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGcpServicesRead,
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

func dataSourceGcpServicesRead(d *schema.ResourceData, meta interface{}) error {
	services := make([]map[string]interface{}, len(integration.GcpServiceNames))
	i := 0
	for k := range integration.GcpServiceNames {
		services[i] = map[string]interface{}{
			"name": k,
		}
		i++
	}
	d.SetId("GCP")
	d.Set("services", services)
	return nil
}
