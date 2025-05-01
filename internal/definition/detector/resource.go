// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	ResourceName = "signalfx_detector"
	AppPath      = "/detector/v2"
)

func NewResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,
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
		CustomizeDiff: customdiff.If(resourceValidateCond, resourceValidateFunc),
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
		Name:              dt.Name,
		AuthorizedWriters: dt.AuthorizedWriters,
		Description:       dt.Description,
		TimeZone:          dt.TimeZone,
		MaxDelay:          dt.MaxDelay,
		MinDelay:          dt.MinDelay,
		ProgramText:       dt.ProgramText,
		Rules:             dt.Rules,
		Tags: common.Unique(
			pmeta.LoadProviderTags(ctx, meta),
			dt.Tags,
		),
		Teams:                pmeta.MergeProviderTeams(ctx, meta, dt.Teams),
		VisualizationOptions: dt.VisualizationOptions,
		ParentDetectorId:     dt.ParentDetectorId,
		DetectorOrigin:       dt.DetectorOrigin,
	})
	if common.HandleError(ctx, err, data) != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	issues = tfext.AppendDiagnostics(issues,
		tfext.AsErrorDiagnostics(
			data.Set("url",
				pmeta.LoadApplicationURL(ctx, meta, AppPath, resp.Id, "edit"),
			),
		)...,
	)

	data.SetId(resp.Id)

	// Some fields are only set from calling a the read operation,
	// so to keep the output consistent, this defers the remaining
	// effort to read to get the data
	return tfext.AppendDiagnostics(
		issues,
		resourceRead(ctx, data, meta)...,
	)
}

func resourceRead(ctx context.Context, data *schema.ResourceData, meta any) (issues diag.Diagnostics) {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	dt, err := client.GetDetector(ctx, data.Id())
	if common.HandleError(ctx, err, data) != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	tflog.Debug(ctx, "Read detector details", tfext.NewLogFields().JSON("detector", dt))

	if dt.OverMTSLimit {
		issues = tfext.AppendDiagnostics(issues, tfext.AsWarnDiagnostics(fmt.Errorf("detector is over mts limit"))...)
	}

	issues = tfext.AppendDiagnostics(issues,
		tfext.AsErrorDiagnostics(
			data.Set("url", pmeta.LoadApplicationURL(ctx, meta, AppPath, dt.Id, "edit")),
		)...,
	)

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
		Name:              dt.Name,
		AuthorizedWriters: dt.AuthorizedWriters,
		Description:       dt.Description,
		TimeZone:          dt.TimeZone,
		MaxDelay:          dt.MaxDelay,
		MinDelay:          dt.MinDelay,
		ProgramText:       dt.ProgramText,
		Rules:             dt.Rules,
		Tags: common.Unique(
			pmeta.LoadProviderTags(ctx, meta),
			dt.Tags,
		),
		Teams:                pmeta.MergeProviderTeams(ctx, meta, dt.Teams),
		VisualizationOptions: dt.VisualizationOptions,
		ParentDetectorId:     dt.ParentDetectorId,
		DetectorOrigin:       dt.DetectorOrigin,
	})
	if common.HandleError(ctx, err, data) != nil {
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
	err = common.HandleError(ctx, client.DeleteDetector(ctx, data.Id()), data)
	return tfext.AsErrorDiagnostics(err)
}

func resourceValidateCond(ctx context.Context, diff *schema.ResourceDiff, _ any) (validate bool) {
	tflog.Debug(ctx, "Checking if program text or rules needed to be updated")
	if _, ok := diff.GetOkExists("rule"); ok {
		validate = true
	}
	if _, ok := diff.GetOkExists("program_text"); ok {
		validate = true
	}
	return validate
}

func resourceValidateFunc(ctx context.Context, diff *schema.ResourceDiff, meta any) error {
	var rules []*detector.Rule
	for _, v := range diff.Get("rule").(*schema.Set).List() {
		data := v.(map[string]any)
		rule := &detector.Rule{
			Description:          data["description"].(string),
			DetectLabel:          data["detect_label"].(string),
			Disabled:             data["disabled"].(bool),
			Severity:             detector.Severity(data["severity"].(string)),
			ParameterizedBody:    data["parameterized_body"].(string),
			ParameterizedSubject: data["parameterized_subject"].(string),
			RunbookUrl:           data["runbook_url"].(string),
			Tip:                  data["tip"].(string),
		}
		if data["reminder_notification"] != nil {
			reminderNotification := convert.ToReminderNotification(data)
			rule.ReminderNotification = reminderNotification
		}
		rules = append(rules, rule)
	}

	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Sending detector payload for validation", tfext.NewLogFields().JSON("content", rules))
	return client.ValidateDetector(ctx, &detector.ValidateDetectorRequestModel{
		Name:        diff.Get("name").(string),
		ProgramText: diff.Get("program_text").(string),
		Rules:       rules,
		Tags: common.Unique(
			pmeta.LoadProviderTags(ctx, meta),
			convert.SchemaListAll(diff.Get("tags"), convert.ToString),
		),
		DetectorOrigin:   diff.Get("detector_origin").(string),
		ParentDetectorId: diff.Get("parent_detector_id").(string),
	})
}
