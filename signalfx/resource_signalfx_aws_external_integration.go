package signalfx

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationAWSExternalResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the integration",
			},
			"external_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The SignalFx-generated AWS external ID to use with an AWS integration.",
			},
			"signalfx_aws_account": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The SignalFx AWS account ID to use with an AWS role.",
			},
		},

		Create: func(d *schema.ResourceData, meta interface{}) error {
			return IntegrationAWSCreate(d, meta, integration.EXTERNAL_ID)
		},
		Read:   IntegrationAWSRead,
		Delete: IntegrationAWSDelete,
		Exists: IntegrationAWSExists,
	}
}
