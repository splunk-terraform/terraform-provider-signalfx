// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts"
)

type dashboardAttributeReader interface {
	GetAttribute(context.Context, path.Path, any) diag.Diagnostics
}

// Read chart objects directly rather than duplicating the curated chart fields
// in native structs. Known siblings remain available when another value is unknown.
func (r *observabilityDashboardResource) model(ctx context.Context, input dashboardAttributeReader) (observabilityDashboardModel, diag.Diagnostics) {
	var model observabilityDashboardModel
	var controls types.Object
	var containers types.List
	var diags diag.Diagnostics
	diags.Append(input.GetAttribute(ctx, path.Root("id"), &model.ID)...)
	diags.Append(input.GetAttribute(ctx, path.Root("title"), &model.Title)...)
	diags.Append(input.GetAttribute(ctx, path.Root("control_bar"), &controls)...)
	diags.Append(input.GetAttribute(ctx, path.Root("container"), &containers)...)
	if diags.HasError() {
		return model, diags
	}
	controlValue, err := controls.ToTerraformValue(ctx)
	if err != nil {
		diags.AddError("Invalid dashboard controls", err.Error())
		return model, diags
	}
	if controlValue.IsFullyKnown() {
		diags.Append(controls.As(ctx, &model.ControlBar, basetypes.ObjectAsOptions{})...)
	}
	for index, value := range containers.Elements() {
		panelPath := path.Root("container").AtListIndex(index)
		if value.IsUnknown() {
			// Keep the position so later entries report the right index.
			model.Container = append(model.Container, dashifyDashboardContainerModel{
				TemplateID: types.StringUnknown(), InlineContent: jsontypes.NewNormalizedUnknown(), ChartUnknown: true,
			})
			continue
		}
		if value.IsNull() {
			diags.AddAttributeError(panelPath, "Invalid panel content", "A container must be a non-null panel object.")
			continue
		}
		attributes := value.(types.Object).Attributes()
		panel := dashifyDashboardContainerModel{
			TemplateID: attributes["template_id"].(types.String), InlineContent: attributes["inline_content"].(jsontypes.Normalized),
		}
		if layout := attributes["layout"].(types.Object); !layout.IsUnknown() {
			diags.Append(layout.As(ctx, &panel.Layout, basetypes.ObjectAsOptions{})...)
		}
		for _, chart := range r.engine.Charts() {
			content := attributes[chart.Name].(types.Object)
			if content.IsNull() {
				continue
			}
			if content.IsUnknown() {
				panel.ChartUnknown = true
				continue
			}
			if panel.Chart.Name != "" {
				diags.AddAttributeError(panelPath, "Invalid panel content", "Configure exactly one chart type, template_id, or inline_content per panel.")
				continue
			}
			panel.Chart = charts.Content{Name: chart.Name, Value: content}
		}
		model.Container = append(model.Container, panel)
	}
	return model, diags
}

func (r *observabilityDashboardResource) setState(ctx context.Context, state *tfsdk.State, model observabilityDashboardModel) diag.Diagnostics {
	attributeTypes := state.Schema.Type().(types.ObjectType).AttrTypes
	controls, diags := types.ObjectValueFrom(ctx, attributeTypes["control_bar"].(types.ObjectType).AttrTypes, model.ControlBar)
	if diags.HasError() {
		return diags
	}
	panelType := attributeTypes["container"].(types.ListType).ElemType.(types.ObjectType)
	panels := make([]attr.Value, 0, len(model.Container))
	for _, panel := range model.Container {
		layout, fields := types.ObjectValueFrom(ctx, panelType.AttrTypes["layout"].(types.ObjectType).AttrTypes, panel.Layout)
		diags.Append(fields...)
		values := map[string]attr.Value{
			"inline_content": panel.InlineContent,
			"template_id":    panel.TemplateID,
			"layout":         layout,
		}
		for _, chart := range r.engine.Charts() {
			objectType := panelType.AttrTypes[chart.Name].(types.ObjectType)
			values[chart.Name] = types.ObjectNull(objectType.AttrTypes)
		}
		if panel.Chart.Name != "" {
			values[panel.Chart.Name] = panel.Chart.Value
		}
		value, fields := types.ObjectValue(panelType.AttrTypes, values)
		diags.Append(fields...)
		panels = append(panels, value)
	}
	containers := types.ListNull(panelType)
	if len(panels) > 0 {
		var fields diag.Diagnostics
		containers, fields = types.ListValue(panelType, panels)
		diags.Append(fields...)
	}
	value, fields := types.ObjectValue(attributeTypes, map[string]attr.Value{
		"id": model.ID, "title": model.Title, "control_bar": controls, "container": containers,
	})
	diags.Append(fields...)
	if !diags.HasError() {
		diags.Append(state.Set(ctx, value)...)
	}
	return diags
}
