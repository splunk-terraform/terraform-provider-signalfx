// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type notificationValidator struct {
	typeRequires map[string]path.Expressions
}

func newNotificationValidator() validator.Object {
	return &notificationValidator{
		typeRequires: map[string]path.Expressions{
			"AmazonEventBridge": {path.MatchRelative().AtName("credential_id")},
			"BigPanda":          {path.MatchRelative().AtName("credential_id")},
			"Email":             {path.MatchRelative().AtName("email")},
			"Jira":              {path.MatchRelative().AtName("credential_id")},
			"Opsgenie":          {path.MatchRelative().AtName("credential_id")},
			"Office365":         {path.MatchRelative().AtName("credential_id")},
			"PagerDuty":         {path.MatchRelative().AtName("credential_id")},
			"ServiceNow":        {path.MatchRelative().AtName("credential_id")},
			"Slack":             {path.MatchRelative().AtName("credential_id"), path.MatchRelative().AtName("channel")},
			"TeamEmail":         {path.MatchRelative().AtName("team")},
			"Team":              {path.MatchRelative().AtName("team")},
			"VictorOps":         {path.MatchRelative().AtName("credential_id"), path.MatchRelative().AtName("routing_key")},
			"Webhook":           {},
			"XMatters":          {path.MatchRelative().AtName("credential_id")},
		},
	}
}

func (v *notificationValidator) Description(ctx context.Context) string {
	return "Ensures that the notification configuration is valid."
}

func (v *notificationValidator) MarkdownDescription(ctx context.Context) string {
	var sb strings.Builder
	_, _ = sb.WriteString("Ensures that the notification configuration is valid.\n\n")
	_, _ = sb.WriteString("The required fields for a notification depend on the value of the `type` field. " +
		"The following table outlines the required fields for each notification type:\n\n")
	_, _ = sb.WriteString("| Notification Type | Required Fields |\n")
	_, _ = sb.WriteString("|-------------------|-----------------|\n")

	for _, notifType := range slices.Sorted(maps.Keys(v.typeRequires)) {
		paths := v.typeRequires[notifType]
		var pathStrs []string
		for _, p := range paths {
			pathStrs = append(pathStrs, "`"+p.String()+"`")
		}
		_, _ = sb.WriteString("| " + notifType + " | " + strings.Join(pathStrs, ", ") + " |\n")
	}

	return sb.String()
}

func (v *notificationValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	var model NotificationModel
	if resp.Diagnostics.Append(req.ConfigValue.As(ctx, &model, basetypes.ObjectAsOptions{})...); resp.Diagnostics.HasError() {
		return
	}

	if model.Type.IsNull() || model.Type.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Invalid Notification Type",
			"The notification type must be set to a valid value before other fields can be validated.",
		)
		return
	}

	// Update the path to include the notification type for more specific diagnostics from the AlsoRequires validator.
	req.Path = req.Path.AtName("type=" + model.Type.ValueString())

	if paths, ok := v.typeRequires[model.Type.ValueString()]; ok && len(paths) > 0 {
		objectvalidator.AlsoRequires(paths...).ValidateObject(ctx, req, resp)
	}

	// Restore the original path for any further validators that may be chained after this one.
	req.Path = req.Path.ParentPath()
}
