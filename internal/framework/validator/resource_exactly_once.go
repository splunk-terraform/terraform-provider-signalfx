// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ExactlyOnce struct {
	values path.Expressions
}

var (
	_ resource.ConfigValidator = (*ExactlyOnce)(nil)
)

func NewResourceExactlyOnce(values ...path.Expression) ExactlyOnce {
	return ExactlyOnce{
		values: path.Expressions(values),
	}
}

func (v ExactlyOnce) Description(ctx context.Context) string {
	return fmt.Sprintf("Exactly one of these attributes can be used together: %v", v)
}
func (v ExactlyOnce) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ExactlyOnce) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if len(v.values) == 0 {
		return
	}
	var configured, unknowns path.Paths

	for _, expr := range v.values {
		matched, details := req.Config.PathMatches(ctx, expr)

		resp.Diagnostics.Append(details...)
		if details.HasError() {
			continue
		}

		for _, p := range matched {
			var value attr.Value
			details = req.Config.GetAttribute(ctx, p, &value)

			resp.Diagnostics.Append(details...)
			if details.HasError() {
				continue
			}

			switch {
			case value.IsNull():
				// Skip null values
			case value.IsUnknown():
				unknowns.Append(p)
			default:
				configured.Append(p)
			}
		}
	}

	if len(configured) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Empty(),
			"Multiple attributes configured",
			fmt.Sprintf("Only one of the following attributes can be configured at a time: %v", configured),
		)
	}

	if len(configured) == 0 && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddAttributeError(
			path.Empty(),
			"No attributes configured",
			fmt.Sprintf("At least one of the following attributes must be configured: %v", v.values),
		)
	}

	if len(unknowns) > 0 {
		resp.Diagnostics.AddAttributeWarning(
			path.Empty(),
			"Unknown attributes configured",
			fmt.Sprintf("The following attributes are unknown: %v", unknowns),
		)
	}
}
