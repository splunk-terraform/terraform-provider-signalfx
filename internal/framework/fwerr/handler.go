// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwerr

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/signalfx/signalfx-go"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

// ErrorHandler abstracts the required error handling logic for the framework API.
// This will standardize how the error is returned to the user.
func ErrorHandler(ctx context.Context, state tfsdk.State, err error) diag.Diagnostics {
	if err == nil {
		return nil
	}

	var info diag.Diagnostics

	sfxerr, ok := signalfx.AsResponseError(err)
	if !ok {
		info.AddError("Issue handling request", err.Error())
		return info
	}

	switch sfxerr.Code() {
	case http.StatusNotFound:
		tflog.Info(ctx,
			"Resource is no longer available, most likely removed manually. "+
				"Remove the current state for provided resource",
		)
		info.AddWarning(err.Error(), sfxerr.Details())
		state.RemoveResource(ctx)
	default:
		info.AddError(err.Error(), sfxerr.Details())
	}

	tflog.Error(ctx, "There was an issue handling request", tfext.NewLogFields().
		Error(err).
		Field("details", sfxerr.Details()),
	)

	return info
}
