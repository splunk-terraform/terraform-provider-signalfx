package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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

		Create: integrationAWSExternalCreate,
		Read:   integrationAWSExternalRead,
		Delete: integrationAWSExternalDelete,
		Exists: integrationAWSExternalExists,
	}
}

func integrationAWSExternalExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

func integrationAWSExternalRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Id())
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
	if err := d.Set("signalfx_aws_account", int.SfxAwsAccountArn); err != nil {
		return err
	}

	return nil
}

func getPayloadAWSExternalIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	// We can't leave this empty, even though we don't need it yet
	aws := &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: integration.EXTERNAL_ID,
		Name:       d.Get("name").(string),
		PollRate:   300000,
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

	int, err := config.Client.CreateAWSCloudWatchIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)
	if err := d.Set("signalfx_aws_account", int.SfxAwsAccountArn); err != nil {
		return err
	}
	if err := d.Set("external_id", int.ExternalId); err != nil {
		return err
	}
	if err := d.Set("name", int.Name); err != nil {
		return err
	}

	// This method does not read back anything from the API except the
	// id and external ID above.
	return nil
}

func integrationAWSExternalDelete(d *schema.ResourceData, meta interface{}) error {
	return DoIntegrationAWSDelete(d, meta)
}

// DoIntegrationAWSDelete is public because it is also used by integrationAWSTokenDelete
func DoIntegrationAWSDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	// Retrieve current integration state
	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error fetching existing integration for integration %s, %s", d.Id(), err.Error())
	}

	// Only disable the Cloudwatch Metric Stream synchronization if needed
	if int.MetricStreamsSyncState != "" && int.MetricStreamsSyncState != "DISABLED" {
		int.MetricStreamsSyncState = "CANCELLING"
		_, err := config.Client.UpdateAWSCloudWatchIntegration(context.TODO(), d.Id(), int)
		if err != nil {
			if strings.Contains(err.Error(), "40") {
				err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
			}
			return err
		}
		// Wait for expected Cloudwatch Metric Streams sync state to be disabled
		if _, err = waitForAWSIntegrationMetricStreamsSyncStateDelete(d, config, int.Id); err != nil {
			return err
		}
	}

	return config.Client.DeleteAWSCloudWatchIntegration(context.TODO(), d.Id())
}

func waitForAWSIntegrationMetricStreamsSyncStateDelete(d *schema.ResourceData, config *signalfxConfig, id string) (*integration.AwsCloudWatchIntegration, error) {
	pending := []string{
		"ENABLED",
		"CANCELLING",
	}
	target := []string{
		"",
		"DISABLED",
	}

	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Refresh: func() (interface{}, string, error) {
			int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), id)
			if err != nil {
				return 0, "", err
			}
			return int, int.MetricStreamsSyncState, nil
		},
		Timeout:    d.Timeout(schema.TimeoutUpdate) - time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	int, err := stateConf.WaitForState()
	if err != nil {
		return nil, fmt.Errorf(
			"Error waiting for integration (%s) Cloudwatch Metric Streams sync state to become disabled: %s",
			id, err)
	}
	return int.(*integration.AwsCloudWatchIntegration), nil
}
