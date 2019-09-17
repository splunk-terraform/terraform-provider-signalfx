package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationAWSExternalResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"external_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The SignalFx-generated AWS account ID to use with an AWS integration.",
			},
		},

		Create: integrationAWSExternalCreate,
		Read:   integrationAWSExternalRead,
		Delete: integrationAWSExternalDelete,
		Exists: integrationAWSExternalExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationAWSExternalExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

func integrationAWSExternalRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	if int.ExternalId != "" {
		if err := d.Set("external_id", int.ExternalId); err != nil {
			return err
		}
	}

	return awsIntegrationAPIToTF(d, int)
}

func getPayloadAWSExternalIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	aws := &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: integration.EXTERNAL_ID,
	}

	return aws, nil
}

func integrationAWSExternalCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAWSExternalIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create AWS External Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateAWSCloudWatchIntegration(payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)
	if err := d.Set("external_id", int.ExternalId); err != nil {
		return err
	}

	// This method does not read back anything from the API except the
	// id and external ID above.
	return nil
}

func integrationAWSExternalDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteAWSCloudWatchIntegration(d.Id())
}
