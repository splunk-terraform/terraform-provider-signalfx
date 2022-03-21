/*
 * Resource for ServiceNow integration
 *
 * https://dev.splunk.com/observability/reference/api/integrations/latest
 * https://docs.splunk.com/Observability/admin/notif-services/servicenow.html
 */

package signalfx

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/signalfx/signalfx-go/integration"
)

const (
	serviceNowIntegrationName = "Service Now"
	serviceNowTypeIncident    = "Incident"
	serviceNowTypeProblem     = "Problem"
)

func integrationServiceNowResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User name used to authenticate the Service Now integration.",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Password used to authenticate the Service Now integration.",
			},
			"instance_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the ServiceNow instance, for example `myInstances.service-now.com`.",
			},
			"issue_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{serviceNowTypeIncident, serviceNowTypeProblem}, false),
				Description:  fmt.Sprintf("The type of issue in standard ITIL terminology. The allowed values are '%s' and '%s'.", serviceNowTypeIncident, serviceNowTypeProblem),
			},
			"alert_triggered_payload_template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An optional template that Observability Cloud uses to create the ServiceNow POST JSON payloads when an alert sends a notification to ServiceNow. Use this optional field to send the values of Observability Cloud alert properties to specific fields in ServiceNow. See API reference for details.",
			},
			"alert_resolved_payload_template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An optional template that Observability Cloud uses to create the ServiceNow PUT JSON payloads when an alert is cleared in ServiceNow. Use this optional field to send the values of Observability Cloud alert properties to specific fields in ServiceNow. See API reference for details.",
			},
		},

		Create: integrationServiceNowCreate,
		Read:   integrationServiceNowRead,
		Update: integrationServiceNowUpdate,
		Delete: integrationServiceNowDelete,
		Exists: integrationServiceNowExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getServiceNowIntegration(d *schema.ResourceData) *integration.ServiceNowIntegration {
	snow := &integration.ServiceNowIntegration{
		Type:         integration.SERVICE_NOW,
		Name:         d.Get("name").(string),
		Enabled:      d.Get("enabled").(bool),
		InstanceName: d.Get("instance_name").(string),
		IssueType:    d.Get("issue_type").(string),
		Username:     d.Get("username").(string),
		Password:     d.Get("password").(string),
	}
	if val, ok := d.GetOk("alert_triggered_payload_template"); ok {
		snow.AlertTriggeredPayloadTemplate = val.(string)
	}
	if val, ok := d.GetOk("alert_resolved_payload_template"); ok {
		snow.AlertResolvedPayloadTemplate = val.(string)
	}
	return snow
}

func setServiceNowIntegration(d *schema.ResourceData, snow *integration.ServiceNowIntegration) error {
	// API doesn't return username and password, so we ignore them.
	if err := d.Set("name", snow.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", snow.Enabled); err != nil {
		return err
	}
	if err := d.Set("instance_name", snow.InstanceName); err != nil {
		return err
	}
	if err := d.Set("issue_type", snow.IssueType); err != nil {
		return err
	}
	if snow.AlertTriggeredPayloadTemplate != "" {
		if err := d.Set("alert_triggered_payload_template", snow.AlertTriggeredPayloadTemplate); err != nil {
			return err
		}
	}
	if snow.AlertResolvedPayloadTemplate != "" {
		if err := d.Set("alert_resolved_payload_template", snow.AlertResolvedPayloadTemplate); err != nil {
			return err
		}
	}
	return nil
}

func integrationServiceNowExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)

	_, err := config.Client.GetServiceNowIntegration(context.TODO(), d.Id())
	return handleIntegrationExists(err)
}

func integrationServiceNowRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	in, err := config.Client.GetServiceNowIntegration(context.TODO(), d.Id())
	if !handleIntegrationRead(err, d) {
		return err
	}
	logIntegrationResponse(in, serviceNowIntegrationName)

	return setServiceNowIntegration(d, in)
}

func integrationServiceNowCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	out := getServiceNowIntegration(d)
	logIntegrationCreateRequest(out, serviceNowIntegrationName)

	in, err := config.Client.CreateServiceNowIntegration(context.TODO(), out)
	if !handleIntegrationChange(err, d, in) {
		return err
	}
	logIntegrationResponse(in, serviceNowIntegrationName)

	return setServiceNowIntegration(d, in)
}

func integrationServiceNowUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	out := getServiceNowIntegration(d)
	logIntegrationUpdateRequest(out, serviceNowIntegrationName)

	in, err := config.Client.UpdateServiceNowIntegration(context.TODO(), d.Id(), out)
	if !handleIntegrationChange(err, d, in) {
		return err
	}
	logIntegrationResponse(in, serviceNowIntegrationName)

	return setServiceNowIntegration(d, in)
}

func integrationServiceNowDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteServiceNowIntegration(context.TODO(), d.Id())
}
