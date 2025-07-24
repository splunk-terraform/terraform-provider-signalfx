// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwerr

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/signalfx/signalfx-go"
)

// ErrorHandler abstracts the required error handling logic for the framework API.
// This will standardise how the error is returned to the user.
func ErrorHandler(err error) diag.Diagnostics {
	if err == nil {
		return nil
	}

	var info diag.Diagnostics
	if sfxerr, ok := signalfx.AsResponseError(err); ok {
		info.AddError(sfxerr.Error(), sfxerr.Details())
	} else {
		info.AddError("Issue handling request", err.Error())
	}

	return info
}
