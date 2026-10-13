// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
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

// Converts the Terraform model into a complete Dashify document and its direct imports.
func buildDashboardSpec(model observabilityDashboardModel) (json.RawMessage, []string, error) {
	spec := map[string]any{"title": model.Title.ValueString()}
	if model.ControlBar != nil {
		spec["controlBar"] = buildDashifyControlBar(model.ControlBar)
	}
	children := make([]any, len(model.Container))
	items := make([]any, len(model.Container))
	var imports []string
	for i := range model.Container {
		id := fmt.Sprintf("_.%d", i)
		child, reference, err := buildDashifyPanelChild(spec, model.Container[i], id)
		if err != nil {
			return nil, nil, fmt.Errorf("container %s template: %w", id, err)
		}
		children[i] = child
		if reference != "" {
			imports = append(imports, reference)
		}
		item, err := buildDashifyLayoutItem(id, model.Container[i].Layout)
		if err != nil {
			return nil, nil, fmt.Errorf("container %s layout: %w", id, err)
		}
		items[i] = item
	}
	spec[dashifyDashboardElement] = children
	spec["layout"] = map[string]any{"saved": map[string]any{"_": map[string]any{"items": items}}}
	raw, err := json.Marshal(spec)
	return raw, imports, err
}

// Wraps an imported Template or opaque inline content in a Panel; only
// imports add API metadata, signaled by a non-empty reference.
func buildDashifyPanelChild(spec map[string]any, model dashifyDashboardContainerModel, id string) (map[string]any, string, error) {
	if model.TemplateID.IsUnknown() || model.InlineContent.IsUnknown() {
		return nil, "", fmt.Errorf("template_id and inline_content must be known before writing")
	}
	idSet := !model.TemplateID.IsNull()
	contentSet := !model.InlineContent.IsNull()
	if idSet == contentSet {
		return nil, "", fmt.Errorf("exactly one of template_id or inline_content must be set")
	}
	if contentSet {
		content, err := decodeDashifyInlineContent(model.InlineContent.ValueString())
		if err != nil {
			return nil, "", err
		}
		return map[string]any{dashifyPanelElement: []any{content}}, "", nil
	}
	if !fwdashify.ObservabilityTemplateIDValid(model.TemplateID.ValueString()) {
		return nil, "", fmt.Errorf("template_id must be a Template ID without an API path")
	}
	alias := dashifyImportAlias(id)
	reference := fwdashify.ObservabilityTemplateReference(model.TemplateID.ValueString())
	spec[dashifyImportPrefix+alias] = reference
	return map[string]any{
		dashifyPanelElement: []any{
			map[string]any{dashifyImportElement + alias + ">": []any{}},
		},
	}, reference, nil
}

// Validates an opaque Panel child and reserves import elements for template_id.
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
		return nil, fmt.Errorf("inline_content cannot be an import element; use template_id instead")
	}
	if nestedTag, ok := findDashifyImportElement(content); ok {
		return nil, fmt.Errorf("inline_content cannot contain import element %q; use template_id instead", nestedTag)
	}
	return content, nil
}

// Derives a stable import alias from a container's positional layout ID.
func dashifyImportAlias(id string) string {
	return "widget" + strings.ReplaceAll(strings.TrimPrefix(id, "_."), ".", "_")
}

// Consumes modeled fields and warns about leftovers that a complete-document update would drop.
func parseDashboardTemplate(record *template.Template) (observabilityDashboardModel, diag.Diagnostics) {
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
			// Retain the update guard when a placeholder still carries layout data.
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
		containerPath := fmt.Sprintf("container.%d", len(model.Container))
		panel, panelLeftovers, err := parseDashifyPanel(spec, used, id, containerPath, value)
		if err != nil {
			return model, formatDashboardParseError(fmt.Errorf("container %s: %w", id, err))
		}
		delete(node, tag)
		panel.Layout = layouts[i]
		model.Container = append(model.Container, panel)
		leftovers = append(leftovers, dashifyLeftovers(containerPath, node)...)
		leftovers = append(leftovers, panelLeftovers...)
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
				"The stored dashboard contains %d field(s) this provider does not model:\n  %s\n\nThese fields are absent from Terraform state, so this resource blocks updates that would remove them. To manage this dashboard, import it as signalfx_observability_template with its complete specification instead.",
				len(leftovers),
				strings.Join(leftovers, "\n  "),
			),
		)
	}
	return model, diags
}

// Resolves imports to Template IDs and preserves other elements as opaque inline content.
func parseDashifyPanel(spec map[string]any, used map[string]bool, id, path string, raw any) (dashifyDashboardContainerModel, []string, error) {
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
	panelContent := unwrapDashifyChartContent(content)
	tag, value, err := oneDashifyElement(panelContent)
	if err != nil {
		return dashifyDashboardContainerModel{}, nil, err
	}
	if !isDashifyImportElement(tag) {
		if nestedTag, found := findDashifyImportElement(content); found {
			return dashifyDashboardContainerModel{}, nil, fmt.Errorf("container %s contains unsupported nested import element %q", id, nestedTag)
		}
		encoded, err := json.Marshal(content)
		if err != nil {
			return dashifyDashboardContainerModel{}, nil, fmt.Errorf("container %s encode inline content: %w", id, err)
		}
		return dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(string(encoded))}, nil, nil
	}
	alias := strings.TrimSuffix(strings.TrimPrefix(tag, dashifyImportElement), ">")
	reference, ok := spec[dashifyImportPrefix+alias].(string)
	if !ok {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("import %q has no matching declaration", alias)
	}
	idValue, ok := fwdashify.ObservabilityTemplateIDFromReference(reference)
	if !ok {
		return dashifyDashboardContainerModel{}, nil, fmt.Errorf("import %q references %q rather than a Template ID", alias, reference)
	}
	used[alias] = true
	// Import elements normally carry an empty argument list. If they carry
	// anything else, leave the tag behind so Read warns that an update drops it.
	if args, ok := value.([]any); ok && len(args) == 0 {
		delete(panelContent, tag)
	}
	return dashifyDashboardContainerModel{TemplateID: types.StringValue(idValue)}, dashifyLeftovers(path, panelContent), nil
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

func unwrapDashifyChartContent(content map[string]any) map[string]any {
	tag, value, err := oneDashifyElement(content)
	if err != nil || tag != dashifyChartElement || len(content) != 1 {
		return content
	}
	items, ok := value.([]any)
	if !ok || len(items) != 1 {
		return content
	}
	inner, ok := items[0].(map[string]any)
	if !ok {
		return content
	}
	if _, _, err := oneDashifyElement(inner); err != nil {
		return content
	}
	return inner
}
