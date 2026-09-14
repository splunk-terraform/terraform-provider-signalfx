// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateObservabilityTitle(resp *resource.ValidateConfigResponse, titlePath path.Path, title types.String) {
	if title.IsUnknown() {
		return
	}
	if title.IsNull() || strings.TrimSpace(title.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(titlePath, "Missing required value", "title must contain at least one non-whitespace character")
	}
}

func nonEmptyStringListValidators() []validator.List {
	return []validator.List{
		listvalidator.NoNullValues(),
		listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
	}
}
