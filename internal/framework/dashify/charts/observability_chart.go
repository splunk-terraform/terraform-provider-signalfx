// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package charts

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
)

const observabilityChartElement = "<Chart>"

// Chart attributes are discovered from the curated definitions. Keep their
// Terraform values intact so that unknown nested values survive planning.
type observabilityChartModel struct {
	ID, Title types.String
	Name      string
	Content   types.Object
}

type observabilityChartAttributeReader interface {
	GetAttribute(context.Context, path.Path, any) diag.Diagnostics
}

func (r *observabilityChartResource) model(ctx context.Context, input observabilityChartAttributeReader) (observabilityChartModel, diag.Diagnostics) {
	var model observabilityChartModel
	var diags diag.Diagnostics
	diags.Append(input.GetAttribute(ctx, path.Root("id"), &model.ID)...)
	diags.Append(input.GetAttribute(ctx, path.Root("title"), &model.Title)...)
	for _, chart := range r.engine.Charts() {
		var content types.Object
		diags.Append(input.GetAttribute(ctx, path.Root(chart.Name), &content)...)
		if content.IsNull() {
			continue
		}
		if model.Name != "" {
			diags.AddError("Multiple chart types", "Configure exactly one chart type.")
			return model, diags
		}
		model.Name, model.Content = chart.Name, content
	}
	if model.Name == "" {
		diags.AddError("Missing chart content", "Configure exactly one chart type.")
	}
	return model, diags
}

func (r *observabilityChartResource) write(ctx context.Context, model observabilityChartModel) (*template.CreateUpdateTemplateRequest, diag.Diagnostics) {
	if model.Title.IsUnknown() {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unknown chart title", "title must be known before writing a Chart Template.")}
	}
	content, diags := EncodeContent(ctx, r.engine, Content{Name: model.Name, Value: model.Content})
	if diags.HasError() {
		return nil, diags
	}
	spec, err := json.Marshal(content)
	if err != nil {
		diags.AddError("Invalid chart specification", fmt.Sprintf("Encoding chart specification: %v", err))
		return nil, diags
	}
	root := template.RootElementChart
	return &template.CreateUpdateTemplateRequest{
		Type: fwdashify.ObservabilityTemplateRecordType, Title: model.Title.ValueString(), Spec: spec,
		Metadata: template.WriteMetadata{RootElement: &root},
	}, diags
}

func observabilityChartRecordFromResult(result *template.Result, expectedID string) (*template.Template, diag.Diagnostics) {
	record, err := fwdashify.ObservabilityTemplateFromResult(result)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Chart API response", err.Error())}
	}
	if !fwdashify.ObservabilityTemplateIDValid(record.ID) || expectedID != "" && record.ID != expectedID {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Chart API response", fmt.Sprintf("Template API returned record %q for Chart %q.", record.ID, expectedID))}
	}
	return record, nil
}

func (r *observabilityChartResource) modelFromRecord(ctx context.Context, record *template.Template, prior observabilityChartModel) (observabilityChartModel, diag.Diagnostics) {
	var model observabilityChartModel
	var diags diag.Diagnostics
	if record == nil || record.Metadata == nil || record.Metadata.RootElement == nil || *record.Metadata.RootElement != template.RootElementChart {
		diags.AddError("Unsupported Chart Template", "The Template must have Chart root element metadata; use signalfx_observability_template for other Template types.")
		return model, diags
	}
	content, leftovers, contentDiags := DecodeContent(ctx, r.engine, record.Spec, Content{Name: prior.Name, Value: prior.Content})
	diags.Append(contentDiags...)
	if diags.HasError() {
		return model, diags
	}
	model.Name, model.Content = content.Name, content.Value
	model.ID, model.Title = types.StringValue(record.ID), types.StringValue(record.Title)
	if len(leftovers) > 0 {
		sort.Strings(leftovers)
		diags.AddWarning("Unmodeled Chart fields", fmt.Sprintf(
			"The stored chart contains %d field(s) this provider does not model:\n  %s\n\nThese fields are absent from Terraform state. Any update, including a title-only change, replaces the complete Chart document and will discard them. To preserve these fields, manage this chart as signalfx_observability_template with its complete specification instead.",
			len(leftovers), strings.Join(leftovers, "\n  "),
		))
	}
	return model, diags
}

// Read reports server values outside the configuration rules as one warning.
// Write responses skip this because state comes from the validated plan.
func (r *observabilityChartResource) ruleWarning(model observabilityChartModel) diag.Diagnostics {
	violations := r.engine.Validate(model.Name, model.Content)
	if len(violations) == 0 {
		return nil
	}
	details := make([]string, len(violations))
	for i, violation := range violations {
		details[i] = violation.Detail()
	}
	return diag.Diagnostics{diag.NewWarningDiagnostic("Stored chart violates configuration rules", fmt.Sprintf(
		"The stored chart has %d value(s) that Terraform configuration does not allow:\n  %s\n\nThe chart was read into state. Its configuration must satisfy these rules before it can be written.",
		len(details), strings.Join(details, "\n  "),
	))}
}

func (r *observabilityChartResource) setState(ctx context.Context, state *tfsdk.State, model observabilityChartModel) diag.Diagnostics {
	values := map[string]attr.Value{"id": model.ID, "title": model.Title}
	attributeTypes := state.Schema.Type().(types.ObjectType).AttrTypes
	for _, chart := range r.engine.Charts() {
		objectType := attributeTypes[chart.Name].(types.ObjectType)
		values[chart.Name] = types.ObjectNull(objectType.AttrTypes)
		if chart.Name == model.Name {
			values[chart.Name] = model.Content
		}
	}
	value, diags := types.ObjectValue(attributeTypes, values)
	if !diags.HasError() {
		diags.Append(state.Set(ctx, value)...)
	}
	return diags
}
