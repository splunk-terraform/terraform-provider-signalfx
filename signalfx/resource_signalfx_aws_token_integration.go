package signalfx

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationAWSTokenResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the integration",
			},
			"token_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The SignalFx-generated AWS token to use with an AWS integration.",
			},
			"signalfx_aws_account": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The Splunk Observability AWS account ID to use with an AWS role.",
			},
		},

		Create: func(d *schema.ResourceData, meta interface{}) error {
			return IntegrationAWSCreate(d, meta, integration.SECURITY_TOKEN)
		},
		Read:   IntegrationAWSRead,
		Delete: IntegrationAWSDelete,
		Exists: IntegrationAWSExists,
	}
}
