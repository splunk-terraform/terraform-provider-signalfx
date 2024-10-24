// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

// OnError handles the general case when the signalfx api returns
// an error, and it uses that information to determine what needs to happen.
// This will ensure that the state is cleaned up given the error condition.
// To help simplify error handling, it will always return the error provided.
func OnError(ctx context.Context, err error, data *schema.ResourceData) error {
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
	return err
}
