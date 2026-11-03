// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

const (
	dashifyDashboardElement       = "<Dashboard>"
	dashifyChartElement           = "<Chart>"
	dashifyPanelElement           = "<Panel>"
	dashifySectionElement         = "<Section>"
	dashifyGroupElement           = "<Group>"
	dashifyImportPrefix           = fwdashify.DashifyImportNamespace + ":"
	dashifyImportElement          = "<" + fwdashify.DashifyImportNamespace + "."
	dashifyUnmodeledFieldsSummary = "Dashboard fields not represented in Terraform state"
)

// isDashifyImportElement reports whether an element tag is a template
// import, so the write path (decodeDashifyInlineContent) and the read path
// (parseDashifyPanel) recognize the same tag shape and cannot drift apart.
func isDashifyImportElement(tag string) bool {
	if !strings.HasPrefix(tag, dashifyImportElement) || !strings.HasSuffix(tag, ">") {
		return false
	}
	alias := tag[len(dashifyImportElement) : len(tag)-1]
	return alias != "" && !strings.Contains(alias, ">")
}

// Inline panels embed literal content; only explicit references add imports.
func buildDashboardSpec(ctx context.Context, engine *interpreter.Engine, model observabilityDashboardModel) (json.RawMessage, []string, error) {
	spec := map[string]any{"title": model.Title.ValueString()}
	if model.ControlBar != nil {
		spec["controlBar"] = buildDashifyControlBar(model.ControlBar)
	}
	children := make([]any, len(model.Container))
	items := make([]any, len(model.Container))
	var imports []string
	for i, panel := range model.Container {
		id := fmt.Sprintf("_.%d", i)
		child, err := buildDashifyPanelChild(ctx, engine, panel, dashifyImportAlias(id))
		if err != nil {
			return nil, nil, fmt.Errorf("container %s: %w", id, err)
		}
		if !panel.TemplateID.IsNull() {
			reference := fwdashify.ObservabilityTemplateReference(panel.TemplateID.ValueString())
			spec[dashifyImportPrefix+dashifyImportAlias(id)] = reference
			imports = append(imports, reference)
		}
		children[i] = child
		items[i] = buildDashifyLayoutItem(id, panel.Layout)
	}
	spec[dashifyDashboardElement] = children
	spec["layout"] = map[string]any{"saved": map[string]any{"_": map[string]any{"items": items}}}
	raw, err := json.Marshal(spec)
	return raw, imports, err
}

func buildDashifyPanelChild(ctx context.Context, engine *interpreter.Engine, model dashifyDashboardContainerModel, alias string) (map[string]any, error) {
	if !model.TemplateID.IsNull() {
		return map[string]any{dashifyPanelElement: []any{map[string]any{dashifyImportElement + alias + ">": []any{}}}}, nil
	}
	var content map[string]any
	if !model.InlineContent.IsNull() {
		var err error
		content, err = decodeDashifyInlineContent(model.InlineContent.ValueString())
		if err != nil {
			return nil, err
		}
	} else {
		var diags diag.Diagnostics
		content, diags = charts.EncodeContent(ctx, engine, model.Chart)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid inline chart: %v", diags)
		}
	}
	return map[string]any{dashifyPanelElement: []any{content}}, nil
}

// Validates self-contained raw panel content.
func decodeDashifyInlineContent(raw string) (map[string]any, error) {
	var value any
	if err := decodeDashifyJSON([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("inline_content must be valid JSON: %w", err)
	}
	content, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("inline_content must be a JSON object")
	}
	tag, _, err := oneDashifyElement(content)
	if err != nil {
		return nil, fmt.Errorf("inline_content %w", err)
	}
	if isDashifyImportElement(tag) {
		return nil, fmt.Errorf("inline_content cannot be an import element; use template_id for a shared chart or embed the complete chart content")
	}
	if nestedTag, ok := findDashifyImportElement(content); ok {
		return nil, fmt.Errorf("inline_content cannot contain import element %q; use template_id for a shared chart or embed the complete chart content", nestedTag)
	}
	if key, found := findDashifyImportDependency(content); found {
		return nil, fmt.Errorf("inline_content cannot contain import dependency %q; use template_id for a shared chart or embed the complete chart content", key)
	}
	return content, nil
}

// Derives a stable import alias from a container's positional layout ID.
func dashifyImportAlias(id string) string {
	return "widget" + strings.ReplaceAll(strings.TrimPrefix(id, "_."), ".", "_")
}

// Consumes modeled fields and warns about leftovers that a complete-document update would drop.
func parseDashboardTemplate(ctx context.Context, engine *interpreter.Engine, record *template.Template, prior observabilityDashboardModel, resolve dashboardChartResolver) (observabilityDashboardModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var model observabilityDashboardModel
	if record == nil || record.Metadata == nil || record.Metadata.RootElement == nil {
		return model, unsupportedDashboardSpec("the template has no Dashboard root element metadata")
	}
	if *record.Metadata.RootElement != template.RootElementDashboard {
		return model, unsupportedDashboardSpec(fmt.Sprintf("the template root element is %q, not %q", *record.Metadata.RootElement, template.RootElementDashboard))
	}

	var spec map[string]any
	if err := decodeDashifyJSON(record.Spec, &spec); err != nil || spec == nil {
		if err == nil {
			err = fmt.Errorf("spec is not a JSON object")
		}
		return model, formatDashboardParseError(err)
	}
	model.Title = types.StringValue(record.Title)
	delete(spec, "title")
	controlBar, controlLeftovers, err := parseDashifyControlBar(spec)
	if err != nil {
		return model, formatDashboardParseError(err)
	}
	model.ControlBar = controlBar
	children, ok := spec[dashifyDashboardElement].([]any)
	if !ok {
		return model, unsupportedDashboardSpec(fmt.Sprintf("spec has no %q element list", dashifyDashboardElement))
	}
	delete(spec, dashifyDashboardElement)
	layouts, layoutLeftovers, err := parseDashifyLayouts(spec, len(children))
	if err != nil {
		return model, formatDashboardParseError(err)
	}
	used := map[string]bool{}
	leftovers := layoutLeftovers
	for i, child := range children {
		id := fmt.Sprintf("_.%d", i)
		if child == nil {
			// UI deletion leaves a null child and removes its layout entry. Those
			// placeholders have no rendered content and can be compacted safely.
			// Report layout data that will be discarded with an omitted placeholder.
			if layouts[i] != nil {
				leftovers = append(leftovers, fmt.Sprintf("container.%d (null element; matching layout %s omitted)", i, id))
			}
			continue
		}
		node, ok := child.(map[string]any)
		if !ok {
			return model, formatDashboardParseError(fmt.Errorf("container %s is %T rather than an element object", id, child))
		}
		tag, value, err := oneDashifyElement(node)
		if err != nil {
			return model, formatDashboardParseError(fmt.Errorf("container %s: %w", id, err))
		}
		if tag == dashifySectionElement || tag == dashifyGroupElement {
			return model, unsupportedDashboardSpec(fmt.Sprintf("container %s contains %s; nested dashboards are supported by signalfx_observability_template", id, tag))
		}
		if tag != dashifyPanelElement {
			return model, unsupportedDashboardSpec(fmt.Sprintf("container %s contains unsupported %s; this resource supports flat panels", id, tag))
		}
		position := len(model.Container)
		var previous dashifyDashboardContainerModel
		if position < len(prior.Container) {
			previous = prior.Container[position]
		}
		containerPath := fmt.Sprintf("container.%d", position)
		panel, panelDiags, err := parseDashifyPanel(ctx, engine, spec, used, containerPath, value, previous, resolve)
		diags.Append(panelDiags...)
		if err != nil {
			return model, formatDashboardParseError(fmt.Errorf("container %s: %w", id, err))
		}
		delete(node, tag)
		panel.Layout = layouts[i]
		model.Container = append(model.Container, panel)
		leftovers = append(leftovers, dashifyLeftovers(containerPath, node)...)
	}
	leftovers = append(controlLeftovers, leftovers...)
	for alias := range used {
		delete(spec, dashifyImportPrefix+alias)
	}
	leftovers = append(leftovers, dashifyLeftovers("", spec)...)
	if len(leftovers) > 0 {
		sort.Strings(leftovers)
		diags.AddWarning(
			dashifyUnmodeledFieldsSummary,
			fmt.Sprintf(
				"The stored dashboard contains %d field(s) this provider does not model:\n  %s\n\nThese fields are absent from Terraform state. Any update, including a title-only change, replaces the complete dashboard document and will discard them. To preserve these fields, manage this dashboard as signalfx_observability_template with its complete specification instead.",
				len(leftovers),
				strings.Join(leftovers, "\n  "),
			),
		)
	}
	return model, diags
}

// Expand referenced source charts into literal content, then choose a lossless
// typed representation. Configured raw panels remain raw on refresh.
func parseDashifyPanel(ctx context.Context, engine *interpreter.Engine, spec map[string]any, used map[string]bool, panelPath string, raw any, prior dashifyDashboardContainerModel, resolve dashboardChartResolver) (dashifyDashboardContainerModel, diag.Diagnostics, error) {
	items, ok := raw.([]any)
	if !ok {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("panel value is %T rather than a list", raw)
	}
	if len(items) != 1 {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("panel contains %d elements; exactly one template reference or inline content object is supported", len(items))
	}
	content, ok := items[0].(map[string]any)
	if !ok {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("panel content is %T rather than an element object", items[0])
	}
	if !prior.TemplateID.IsNull() && !prior.TemplateID.IsUnknown() {
		panelContent := content
		if inner, ok := wrappedDashifyChartImport(content); ok && len(content) == 1 {
			panelContent = inner
		}
		tag, value, err := oneDashifyElement(panelContent)
		if err != nil {
			return dashifyDashboardContainerModel{}, nil, err
		}
		if isDashifyImportElement(tag) {
			alias := strings.TrimSuffix(strings.TrimPrefix(tag, dashifyImportElement), ">")
			reference, ok := spec[dashifyImportPrefix+alias].(string)
			if !ok {
				return dashifyDashboardContainerModel{}, nil, fmt.Errorf("import %q has no matching declaration", alias)
			}
			id, ok := fwdashify.ObservabilityTemplateIDFromReference(reference)
			if !ok {
				return dashifyDashboardContainerModel{}, nil, fmt.Errorf("import %q references %q rather than a Template ID", alias, reference)
			}
			used[alias] = true
			if args, ok := value.([]any); ok && len(args) == 0 {
				delete(panelContent, tag)
			}
			var warnings diag.Diagnostics
			if leftovers := dashifyLeftovers(panelPath, panelContent); len(leftovers) > 0 {
				sort.Strings(leftovers)
				warnings.AddWarning(dashifyUnmodeledFieldsSummary, "The shared chart reference contains fields absent from Terraform state: "+strings.Join(leftovers, ", ")+". Any update replaces the complete dashboard document and will discard them.")
			}
			return dashifyDashboardContainerModel{TemplateID: types.StringValue(id)}, warnings, nil
		}
	}
	_, imported := findDashifyImportElement(content)
	content, err := expandDashboardChartContent(content, spec, used, resolve, map[string]bool{})
	if err != nil {
		return dashifyDashboardContainerModel{}, nil, err
	}
	encoded, err := json.Marshal(content)
	if err != nil {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("encode inline content: %w", err)
	}
	panel := dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(string(encoded))}
	if !prior.InlineContent.IsNull() {
		return panel, nil, nil
	}
	chart, leftovers, diags := charts.DecodeContent(ctx, engine, encoded, prior.Chart)
	if !diags.HasError() && len(leftovers) == 0 {
		// A typed import must reproduce the complete document. This also catches
		// explicit nulls and numbers that cannot survive the typed conversion.
		rebuilt, encodeDiags := charts.EncodeContent(ctx, engine, chart)
		if !encodeDiags.HasError() {
			candidate, err := json.Marshal(rebuilt)
			if err == nil {
				equal, compareDiags := panel.InlineContent.StringSemanticEquals(ctx, jsontypes.NewNormalizedValue(string(candidate)))
				if !compareDiags.HasError() && equal {
					return dashifyDashboardContainerModel{Chart: chart}, nil, nil
				}
			}
		}
	}
	if imported {
		reason := "the chart has unsupported content or values that the typed attributes cannot represent"
		if len(leftovers) > 0 {
			sort.Strings(leftovers)
			reason = "the chart has unmodeled fields: " + strings.Join(leftovers, ", ")
		}
		return panel, diag.Diagnostics{diag.NewWarningDiagnostic("Chart preserved as raw inline content", "The complete source chart was kept in inline_content because "+reason+". No source chart record was changed.")}, nil
	}
	return panel, nil, nil
}

// Finds one angle-bracket element while leaving ordinary sibling properties untouched.
func oneDashifyElement(node map[string]any) (string, any, error) {
	var tag string
	var value any
	for key, raw := range node {
		if strings.HasPrefix(key, "<") {
			if tag != "" {
				return "", nil, fmt.Errorf("has multiple dashboard element keys")
			}
			tag = key
			value = raw
		}
	}
	if tag == "" {
		return "", nil, fmt.Errorf("has no dashboard element key")
	}
	return tag, value, nil
}

// Returns unconsumed leaf paths, treating arrays and empty objects as values.
// Modeled objects are removed by their parsers before this function is called.
func dashifyLeftovers(prefix string, node map[string]any) []string {
	var leftovers []string
	for key, value := range node {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if child, ok := value.(map[string]any); ok {
			if len(child) == 0 {
				leftovers = append(leftovers, path)
			} else {
				leftovers = append(leftovers, dashifyLeftovers(path, child)...)
			}
		} else {
			leftovers = append(leftovers, path)
		}
	}
	return leftovers
}

func findDashifyImportElement(value any) (string, bool) {
	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if isDashifyImportElement(key) {
				return key, true
			}
			if tag, ok := findDashifyImportElement(value[key]); ok {
				return tag, true
			}
		}
	case []any:
		for _, child := range value {
			if tag, ok := findDashifyImportElement(child); ok {
				return tag, true
			}
		}
	}
	return "", false
}
