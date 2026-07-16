// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/notification"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

type notificationStringListValidator struct{}

var _ validator.List = notificationStringListValidator{}

// NotificationStringListValidator validates the provider's legacy comma-delimited
// notification representation while it remains part of a resource schema.
func NotificationStringListValidator() validator.List {
	return notificationStringListValidator{}
}

func (notificationStringListValidator) Description(context.Context) string {
	return "each value must be a valid comma-delimited notification destination"
}

func (v notificationStringListValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (notificationStringListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var values []string
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for index, value := range values {
		if _, err := common.NewNotificationFromString(value); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(index),
				"Invalid notification destination",
				err.Error(),
			)
		}
	}
}

// NotificationStringsToAPI converts a Framework list into the SignalFx API model.
func NotificationStringsToAPI(ctx context.Context, value types.List) ([]*notification.Notification, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsNull() {
		return nil, diagnostics
	}

	var values []string
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	items := make([]*notification.Notification, len(values))
	for index, item := range values {
		parsed, err := common.NewNotificationFromString(item)
		if err != nil {
			diagnostics.AddError("Invalid notification destination", err.Error())
			return nil, diagnostics
		}
		items[index] = parsed
	}
	return items, diagnostics
}

// NotificationStringsFromAPI converts API notifications into Framework state.
// An omitted API list preserves a null Terraform value, matching the SDK resource.
func NotificationStringsFromAPI(ctx context.Context, current types.List, items []*notification.Notification) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if len(items) == 0 && current.IsNull() {
		return current, diagnostics
	}

	values, err := common.NewNotificationStringList(items)
	if err != nil {
		diagnostics.AddError("Unable to read notification destinations", err.Error())
		return current, diagnostics
	}

	result, valueDiagnostics := types.ListValueFrom(ctx, types.StringType, values)
	diagnostics.Append(valueDiagnostics...)
	return result, diagnostics
}
