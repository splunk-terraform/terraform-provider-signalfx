// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// Keep the stored JSON when a config change only alters whitespace or key
// order. Without this, importing the API's compact JSON and using a readable
// heredoc in Terraform produces a spurious in-place update.
type observabilityTemplateSpecSemanticPlanModifier struct{}

func (observabilityTemplateSpecSemanticPlanModifier) Description(context.Context) string {
	return "Preserves the existing specification when the JSON content is unchanged."
}

func (m observabilityTemplateSpecSemanticPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (observabilityTemplateSpecSemanticPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	equal, diags := jsontypes.NewNormalizedValue(req.StateValue.ValueString()).StringSemanticEquals(
		ctx, jsontypes.NewNormalizedValue(req.PlanValue.ValueString()),
	)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() && equal {
		resp.PlanValue = req.StateValue
	}
}
