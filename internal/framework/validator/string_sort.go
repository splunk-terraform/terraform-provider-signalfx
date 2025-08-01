// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwvalidator

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SortString struct{}

var _ validator.String = (*SortString)(nil)

func NewSortString() validator.String {
	return &SortString{}
}

func (SortString) Description(_ context.Context) string {
	return "Must provide a property to sort on with the prefix of + or - to defined ascending or descending order."
}

func (s SortString) MarkdownDescription(_ context.Context) string {
	return strings.Join(
		[]string{
			s.Description(context.Background()),
			"Example: `+property` for ascending order or `-property` for descending order.",
		},
		"\n",
	)
}

func (s SortString) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var value types.String
	if details := req.Config.GetAttribute(ctx, req.Path, &value); details.HasError() {
		resp.Diagnostics.Append(details...)
		return
	}
	v := value.ValueString()
	if !strings.HasPrefix(v, "+") && !strings.HasPrefix(v, "-") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Sort String",
			"Sort string must start with either '+' or '-' to indicate ascending or descending order.",
		)
		return
	}
	v = v[1:] // Remove the prefix
	if v == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Empty Sort String",
			"Sort string cannot be empty after the prefix.",
		)
	}
}
