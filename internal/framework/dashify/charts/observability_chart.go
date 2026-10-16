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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
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
	var resp resource.ValidateConfigResponse
	fwdashify.ValidateObservabilityTitle(&resp, path.Root("title"), model.Title)
	if model.Title.IsUnknown() {
		resp.Diagnostics.AddError("Unknown chart title", "title must be known before writing a Chart Template.")
	}
	if resp.Diagnostics.HasError() {
		return nil, resp.Diagnostics
	}
	content, diags := r.engine.Encode(ctx, model.Name, model.Content)
	if diags.HasError() {
		return nil, diags
	}
	spec, err := json.Marshal(map[string]any{observabilityChartElement: []any{content}})
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

func (r *observabilityChartResource) fromResult(ctx context.Context, result *template.Result, expectedID string, prior observabilityChartModel) (observabilityChartModel, diag.Diagnostics) {
	record, err := fwdashify.ObservabilityTemplateFromResult(result)
	if err != nil {
		return observabilityChartModel{}, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Chart API response", err.Error())}
	}
	if !fwdashify.ObservabilityTemplateIDValid(record.ID) || expectedID != "" && record.ID != expectedID {
		return observabilityChartModel{}, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Chart API response", fmt.Sprintf("Template API returned record %q for Chart %q.", record.ID, expectedID))}
	}
	return r.modelFromRecord(ctx, record, prior)
}

func (r *observabilityChartResource) modelFromRecord(ctx context.Context, record *template.Template, prior observabilityChartModel) (observabilityChartModel, diag.Diagnostics) {
	var model observabilityChartModel
	var diags diag.Diagnostics
	if record == nil || record.Metadata == nil || record.Metadata.RootElement == nil || *record.Metadata.RootElement != template.RootElementChart {
		diags.AddError("Unsupported Chart Template", "The Template must have Chart root element metadata; use signalfx_observability_template for other Template types.")
		return model, diags
	}
	name, content, rootLeftovers, err := r.parseSpec(record.Spec)
	if err != nil {
		diags.AddError("Unsupported Chart Template", err.Error())
		return model, diags
	}
	previous := types.ObjectNull(nil)
	if prior.Name == name {
		previous = prior.Content
	}
	var leftovers []string
	model.Content, leftovers, diags = r.engine.Decode(ctx, name, content, previous)
	if diags.HasError() {
		return model, diags
	}
	model.Name, model.ID, model.Title = name, types.StringValue(record.ID), types.StringValue(record.Title)
	leftovers = append(leftovers, rootLeftovers...)
	if len(leftovers) > 0 {
		sort.Strings(leftovers)
		diags.AddWarning("Unmodeled Chart fields", fmt.Sprintf(
			"The stored chart contains %d field(s) this provider does not model:\n  %s\n\nThese fields are absent from Terraform state. Any update, including a title-only change, replaces the complete Chart document and will discard them. To preserve these fields, manage this chart as signalfx_observability_template with its complete specification instead.",
			len(leftovers), strings.Join(leftovers, "\n  "),
		))
	}
	return model, diags
}

func (r *observabilityChartResource) parseSpec(raw []byte) (string, map[string]any, []string, error) {
	spec, err := interpreter.DecodeObject(string(raw))
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid Chart specification: %w", err)
	}
	children, ok := spec[observabilityChartElement].([]any)
	if !ok || len(children) != 1 {
		return "", nil, nil, fmt.Errorf("%s must contain exactly one chart object", observabilityChartElement)
	}
	content, ok := children[0].(map[string]any)
	if !ok {
		return "", nil, nil, fmt.Errorf("%s child is not a chart object", observabilityChartElement)
	}
	var tag string
	for key := range content {
		if strings.HasPrefix(key, "<") {
			if tag != "" {
				return "", nil, nil, fmt.Errorf("chart child has multiple element keys")
			}
			tag = key
		}
	}
	var name string
	for _, chart := range r.engine.Charts() {
		if chart.Element == tag {
			name = chart.Name
			break
		}
	}
	if name == "" {
		return "", nil, nil, fmt.Errorf("chart contains unsupported element %q; use signalfx_observability_template for other chart types", tag)
	}
	if args, valid := content[tag].([]any); !valid || len(args) != 0 {
		return "", nil, nil, fmt.Errorf("%s must have an empty argument list", tag)
	}
	delete(spec, observabilityChartElement)
	return name, content, interpreter.Leftovers("spec", spec), nil
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
