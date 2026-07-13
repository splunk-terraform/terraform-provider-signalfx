// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

/*
 * Resource for BigPanda integration
 *
 * https://dev.splunk.com/observability/reference/api/integrations/latest
 * https://docs.splunk.com/Observability/admin/notif-services/bigpanda.html
 */

package signalfx

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

const bigPandaIntegrationName = "BigPanda"

func integrationBigPandaResource() *schema.Resource {
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
				Description: "Whether the integration is enabled",
			},
			"app_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Application key you get from BigPanda.",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Token you get from BigPanda.",
			},
			"alert_triggered_payload_template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a triggered notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.",
			},
			"alert_resolved_payload_template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a resolved notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.",
			},
		},

		Create: integrationBigPandaCreate,
		Read:   integrationBigPandaRead,
		Update: integrationBigPandaUpdate,
		Delete: integrationBigPandaDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getBigPandaIntegration(d *schema.ResourceData) *integration.BigPandaIntegration {
	bp := &integration.BigPandaIntegration{
		Type:    integration.BIG_PANDA,
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		AppKey:  d.Get("app_key").(string),
		Token:   d.Get("token").(string),
	}
	if val, ok := d.GetOk("alert_triggered_payload_template"); ok {
		bp.AlertTriggeredPayloadTemplate = val.(string)
	}
	if val, ok := d.GetOk("alert_resolved_payload_template"); ok {
		bp.AlertResolvedPayloadTemplate = val.(string)
	}
	return bp
}

func setBigPandaIntegration(d *schema.ResourceData, bp *integration.BigPandaIntegration) error {
	// API doesn't return app_key and token, so we ignore them.
	if err := d.Set("name", bp.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", bp.Enabled); err != nil {
		return err
	}
	if bp.AlertTriggeredPayloadTemplate != "" {
		if err := d.Set("alert_triggered_payload_template", bp.AlertTriggeredPayloadTemplate); err != nil {
			return err
		}
	}
	if bp.AlertResolvedPayloadTemplate != "" {
		if err := d.Set("alert_resolved_payload_template", bp.AlertResolvedPayloadTemplate); err != nil {
			return err
		}
	}
	return nil
}

func integrationBigPandaRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	in, err := config.Client.GetBigPandaIntegration(context.TODO(), d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}
	logIntegrationResponse(in, bigPandaIntegrationName)

	return setBigPandaIntegration(d, in)
}

func integrationBigPandaCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	out := getBigPandaIntegration(d)
	logIntegrationCreateRequest(out, bigPandaIntegrationName)

	in, err := config.Client.CreateBigPandaIntegration(context.TODO(), out)
	if !handleIntegrationChange(err, d, in) {
		return err
	}
	logIntegrationResponse(in, bigPandaIntegrationName)

	return setBigPandaIntegration(d, in)
}

func integrationBigPandaUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	out := getBigPandaIntegration(d)
	logIntegrationUpdateRequest(out, bigPandaIntegrationName)

	in, err := config.Client.UpdateBigPandaIntegration(context.TODO(), d.Id(), out)
	if !handleIntegrationChange(err, d, in) {
		return err
	}
	logIntegrationResponse(in, bigPandaIntegrationName)

	return setBigPandaIntegration(d, in)
}

func integrationBigPandaDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteBigPandaIntegration(context.TODO(), d.Id())
}
