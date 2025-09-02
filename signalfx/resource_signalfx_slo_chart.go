// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sfxgo "github.com/signalfx/signalfx-go"
	chart "github.com/signalfx/signalfx-go/chart"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func sloChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"slo_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the attached SLO",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		CreateContext: slochartCreate,
		ReadContext:   slochartRead,
		UpdateContext: slochartUpdate,
		DeleteContext: slochartDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

/*
Use Resource object to construct json payload in order to create an SLO chart
*/
func getPayloadSloChart(d *schema.ResourceData) *chart.CreateUpdateSloChartRequest {
	return &chart.CreateUpdateSloChartRequest{
		SloId: d.Get("slo_id").(string),
	}
}

func slochartCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*signalfxConfig)
	payload := getPayloadSloChart(d)

	tflog.Debug(ctx, "Creating SLO chart", tfext.NewLogFields().JSON("payload", payload))

	sloChart, err := config.Client.CreateSloChart(ctx, payload)
	if err != nil {
		return diag.FromErr(err)
	}
	// Since things worked, set the URL and move on
	appURL := pmeta.LoadApplicationURL(ctx, meta, CHART_APP_PATH, sloChart.Id)

	if err := d.Set("url", appURL); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sloChart.Id)
	return slochartAPIToTF(d, sloChart)
}

func slochartAPIToTF(d *schema.ResourceData, c *chart.Chart) diag.Diagnostics {
	return diag.FromErr(d.Set("slo_id", c.SloId))
}

func isSlochartNotFound(err error) bool {
	sfxRespErr, ok := err.(*sfxgo.ResponseError)
	return ok && sfxRespErr.Code() == http.StatusNotFound
}

func slochartRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*signalfxConfig)

	sloChart, err := config.Client.GetChart(ctx, d.Id())
	if err != nil {
		if isSlochartNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	appURL := pmeta.LoadApplicationURL(ctx, meta, CHART_APP_PATH, sloChart.Id)

	if err := d.Set("url", appURL); err != nil {
		return diag.FromErr(err)
	}

	return slochartAPIToTF(d, sloChart)
}

func slochartUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*signalfxConfig)
	payload := getPayloadSloChart(d)
	tflog.Debug(ctx, "Updating SLO chart", tfext.NewLogFields().JSON("payload", payload))

	c, err := config.Client.UpdateSloChart(ctx, d.Id(), payload)
	if err != nil {
		return diag.FromErr(err)
	}
	tflog.Debug(ctx, "SLO chart update response", tfext.NewLogFields().JSON("response", c))

	d.SetId(c.Id)
	return slochartAPIToTF(d, c)
}

func slochartDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	config := meta.(*signalfxConfig)

	return diag.FromErr(config.Client.DeleteChart(ctx, d.Id()))
}
