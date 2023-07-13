package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
	"log"
	"strings"
)

func IntegrationAWSExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IntegrationAWSRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	if int.AuthMethod == integration.EXTERNAL_ID && int.ExternalId != "" {
		if err := d.Set("external_id", int.ExternalId); err != nil {
			return err
		}
	}
	if err := d.Set("signalfx_aws_account", int.SfxAwsAccountArn); err != nil {
		return err
	}

	return nil
}

func IntegrationAWSCreate(d *schema.ResourceData, meta interface{}, authMethod integration.AwsAuthMethod) error {
	config := meta.(*signalfxConfig)
	payload, err := getIntegrationPayload(d, authMethod)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create AWS Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateAWSCloudWatchIntegration(context.TODO(), payload)
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
	if authMethod == integration.EXTERNAL_ID {
		if err := d.Set("external_id", int.ExternalId); err != nil {
			return err
		}
	}
	// This method does not read back anything from the API except the few props visible above
	return nil
}

func IntegrationAWSDelete(d *schema.ResourceData, meta interface{}) error {
	return DoIntegrationAWSDelete(d, meta)
}

func getIntegrationPayload(d *schema.ResourceData, authMethod integration.AwsAuthMethod) (*integration.AwsCloudWatchIntegration, error) {
	// We can't leave this empty, even though we don't need it yet
	aws := &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: authMethod,
		Name:       d.Get("name").(string),
		PollRate:   300000,
	}

	return aws, nil
}
