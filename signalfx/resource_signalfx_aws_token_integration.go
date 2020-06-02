package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
				Description: "The SignalFx AWS account ID to use with an AWS role.",
			},
			"named_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
				ForceNew:    true,
			},
		},

		Create: integrationAWSTokenCreate,
		Read:   integrationAWSTokenRead,
		Delete: integrationAWSTokenDelete,
		Exists: integrationAWSTokenExists,
	}
}

func integrationAWSTokenExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationAWSTokenRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return nil
}

func getPayloadAWSTokenIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	// We can't leave this empty, even though we don't need it yet
	defaultPollRate := integration.FiveMinutely
	aws := &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: integration.SECURITY_TOKEN,
		Name:       d.Get("name").(string),
		PollRate:   &defaultPollRate,
	}

	if val, ok := d.GetOk("named_token"); ok {
		aws.NamedToken = val.(string)
	}

	return aws, nil
}

func integrationAWSTokenCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAWSTokenIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create AWS Token Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateAWSCloudWatchIntegration(payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)
	if err := d.Set("name", int.Name); err != nil {
		return err
	}
	if err := d.Set("signalfx_aws_account", int.SfxAwsAccountArn); err != nil {
		return err
	}
	if err := d.Set("named_token", int.NamedToken); err != nil {
		return err
	}

	// This method does not read back anything from the API except the
	// id and Token ID above.
	return nil
}

func integrationAWSTokenDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteAWSCloudWatchIntegration(d.Id())
}
