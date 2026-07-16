// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"time"
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type detectorTimeZoneValidator struct{}

var _ validator.String = detectorTimeZoneValidator{}

func (detectorTimeZoneValidator) Description(context.Context) string {
	return "value must identify a valid IANA time zone"
}

func (v detectorTimeZoneValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (detectorTimeZoneValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.LoadLocation(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid detector time zone", err.Error())
	}
}
