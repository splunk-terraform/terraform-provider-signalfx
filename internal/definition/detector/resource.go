// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	ResourceName = "signalfx_detector"
	AppPath      = "/detector/v2"
)

func NewResource() *schema.Resource {
	return &schema.Resource{
		SchemaFunc:    newSchema,
		CreateContext: resourceCreate,
		ReadContext:   resourceRead,
		UpdateContext: resourceUpdate,
		DeleteContext: resourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		StateUpgraders: []schema.StateUpgrader{
			{Type: v0state().CoreConfigSchema().ImpliedType(), Upgrade: v0stateMigration, Version: 0},
		},
	}
}

func resourceCreate(ctx context.Context, data *schema.ResourceData, meta any) (issues diag.Diagnostics) {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	dt, err := decodeTerraform(data)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	tflog.Debug(ctx, "Creating new detector", tfext.NewLogFields().JSON("detector", dt))

	resp, err := client.CreateDetector(ctx, &detector.CreateUpdateDetectorRequest{
		Name:                 dt.Name,
		AuthorizedWriters:    dt.AuthorizedWriters,
		Description:          dt.Description,
		TimeZone:             dt.TimeZone,
		MaxDelay:             dt.MaxDelay,
		MinDelay:             dt.MinDelay,
		ProgramText:          dt.ProgramText,
		Rules:                dt.Rules,
		Tags:                 dt.Tags,
		Teams:                dt.Teams,
		VisualizationOptions: dt.VisualizationOptions,
		ParentDetectorId:     dt.ParentDetectorId,
		DetectorOrigin:       dt.DetectorOrigin,
	})
	if err != common.OnError(ctx, err, data) {
		return tfext.AsErrorDiagnostics(err)
	}

	issues = tfext.AppendDiagnostics(issues,
		tfext.AsErrorDiagnostics(
			data.Set("url",
				pmeta.LoadApplicationURL(ctx, meta, AppPath, resp.Id, "edit"),
			),
		)...,
	)

	return tfext.AppendDiagnostics(
		issues,
		tfext.AsErrorDiagnostics(encodeTerraform(resp, data))...,
	)
}

func resourceRead(ctx context.Context, data *schema.ResourceData, meta any) (issues diag.Diagnostics) {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	dt, err := client.GetDetector(ctx, data.Id())
	if err != common.OnError(ctx, err, data) {
		return tfext.AsErrorDiagnostics(err)
	}

	tflog.Debug(ctx, "Read detector details", tfext.NewLogFields().JSON("detector", dt))

	if dt.OverMTSLimit {
		issues = tfext.AppendDiagnostics(issues, tfext.AsWarnDiagnostics(fmt.Errorf("detector is over mts limit"))...)
	}

	return tfext.AppendDiagnostics(
		issues,
		tfext.AsErrorDiagnostics(encodeTerraform(dt, data))...,
	)
}

func resourceUpdate(ctx context.Context, data *schema.ResourceData, meta any) (issues diag.Diagnostics) {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	dt, err := decodeTerraform(data)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	tflog.Debug(ctx, "Updating detector", tfext.NewLogFields().
		JSON("detector", dt).
		Field("id", data.Id()),
	)

	resp, err := client.UpdateDetector(ctx, data.Id(), &detector.CreateUpdateDetectorRequest{
		Name:                 dt.Name,
		AuthorizedWriters:    dt.AuthorizedWriters,
		Description:          dt.Description,
		TimeZone:             dt.TimeZone,
		MaxDelay:             dt.MaxDelay,
		MinDelay:             dt.MinDelay,
		ProgramText:          dt.ProgramText,
		Rules:                dt.Rules,
		Tags:                 dt.Tags,
		Teams:                dt.Teams,
		VisualizationOptions: dt.VisualizationOptions,
		ParentDetectorId:     dt.ParentDetectorId,
		DetectorOrigin:       dt.DetectorOrigin,
	})
	if err != common.OnError(ctx, err, data) {
		return tfext.AsErrorDiagnostics(err)
	}

	issues = tfext.AppendDiagnostics(issues,
		tfext.AsErrorDiagnostics(
			data.Set("url", pmeta.LoadApplicationURL(ctx, meta, AppPath, resp.Id, "edit")),
		)...,
	)

	return tfext.AppendDiagnostics(
		issues,
		tfext.AsErrorDiagnostics(encodeTerraform(resp, data))...,
	)
}

func resourceDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}
	err = common.OnError(ctx, client.DeleteDetector(ctx, data.Id()), data)
	return tfext.AsErrorDiagnostics(err)
}
