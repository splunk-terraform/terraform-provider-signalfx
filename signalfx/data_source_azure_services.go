package signalfx

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func dataSourceAzureServices() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAzureServicesRead,
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

func dataSourceAzureServicesRead(d *schema.ResourceData, meta interface{}) error {
	services := make([]map[string]interface{}, len(integration.AzureServiceNames))
	i := 0
	for k := range integration.AzureServiceNames {
		services[i] = map[string]interface{}{
			"name": k,
		}
		i++
	}
	d.SetId("AZURE")
	d.Set("services", services)
	return nil
}
