// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

// HandleError handles the general case when the signalfx api returns
// an error, and it uses that information to determine what needs to happen.
// This will ensure that the state is cleaned up given the error condition.
// To help simplify error handling, it will always return the error provided.
func HandleError(ctx context.Context, err error, data *schema.ResourceData) error {
	re, ok := signalfx.AsResponseError(err)
	if !ok {
		// Not a response error, pass it back
		return err
	}
	switch re.Code() {
	case http.StatusNotFound:
		tflog.Info(ctx, "Resource has been removed externally, removing from state", tfext.NewLogFields().
			Field("route", re.Route()),
		)
		// Clear the id from the state when 404 is returned.
		data.SetId("")
	case http.StatusUnauthorized:
		tflog.Error(ctx, "Token is not authorized", tfext.NewLogFields().
			Field("route", re.Route()).
			Field("code", re.Code()).
			Field("details", re.Details()),
		)
	default:
		tflog.Debug(ctx, "Issue trying to work with the API", tfext.NewLogFields().
			Field("route", re.Route()).
			Field("code", re.Code()).
			Field("details", re.Details()),
		)
	}
	// Preserve original error identity; callers may choose to wrap before calling.
	return err
}

// WrapResponseError augments a signalfx.ResponseError with route/code/details context
// so that error strings surfaced to Terraform users are more actionable, while preserving
// the original error in the chain via %w for errors.Is/As.
func WrapResponseError(err error) error {
	re, ok := signalfx.AsResponseError(err)
	if !ok {
		return err
	}
	details := strings.TrimSpace(re.Details())
	var msg string
	if details != "" {
		msg = fmt.Sprintf("route %q had issues with status code %d: %s", re.Route(), re.Code(), details)
	} else {
		msg = fmt.Sprintf("route %q had issues with status code %d", re.Route(), re.Code())
	}
	return fmt.Errorf("%s: %w", msg, err)
}
