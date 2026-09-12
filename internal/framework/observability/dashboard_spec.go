// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/template"
)

const (
	observabilityDashboardElement = "<Dashboard>"
	observabilityPanelElement     = "<Panel>"
	observabilitySectionElement   = "<Section>"
	observabilityGroupElement     = "<Group>"
	observabilityImportPrefix     = "$import:"
	observabilityImportElement    = "<$import."
	observabilityTemplatePrefix   = "/v2/template/"
	observabilityLeftoverLimit    = 10
)

// buildDashboardSpec converts the Terraform dashboard model into the complete
// Dashify document stored in a Template record and returns its direct imports.
func buildDashboardSpec(model observabilityDashboardModel) (json.RawMessage, []string, error) {
	spec := map[string]any{"title": model.Title.ValueString()}
	if model.ControlBar != nil {
		spec["controlBar"] = buildObservabilityControlBar(model.ControlBar)
	}
	children, saved, imports, err := buildObservabilityContainerList(
		spec,
		observabilityContainersFromDashboardModels(model.Container),
		"_",
		nil,
	)
	if err != nil {
		return nil, nil, err
	}
	spec[observabilityDashboardElement] = children
	layout, err := buildObservabilityLayoutOptions(model.Layout)
	if err != nil {
		return nil, nil, fmt.Errorf("dashboard layout: %w", err)
	}
	layout["saved"] = saved
	spec["layout"] = layout
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal dashboard spec: %w", err)
	}
	return raw, imports, nil
}

// buildObservabilityContainerList recursively builds one Dashify container
// level together with the positional layout entries for that level.
func buildObservabilityContainerList(spec map[string]any, containers []observabilityContainer, listKey string, metadata map[string]any) ([]any, map[string]any, []string, error) {
	children := make([]any, len(containers))
	items := make([]any, len(containers))
	saved := map[string]any{}
	var imports []string

	for i, container := range containers {
		id := fmt.Sprintf("%s.%d", listKey, i)
		// TODO(charts): Serialize generated chart content through its BuildSpec
		// contract and wrap the result as an inline Dashify Panel child.
		switch {
		case container.Section != nil:
			sectionChildren, sectionSaved, sectionImports, err := buildObservabilityContainerList(spec, container.Section.Container, id, buildObservabilitySectionMetadata(container.Section))
			if err != nil {
				return nil, nil, nil, fmt.Errorf("container %s section: %w", id, err)
			}
			child := map[string]any{observabilitySectionElement: sectionChildren}
			layout, err := buildObservabilityLayoutOptions(container.Section.Layout)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("container %s section layout: %w", id, err)
			}
			if len(layout) > 0 {
				child["layout"] = layout
			}
			children[i] = child
			for key, value := range sectionSaved {
				saved[key] = value
			}
			imports = append(imports, sectionImports...)
		case container.Group != nil:
			groupChildren, groupSaved, groupImports, err := buildObservabilityContainerList(spec, container.Group.Container, id, buildObservabilityGroupMetadata(container.Group))
			if err != nil {
				return nil, nil, nil, fmt.Errorf("container %s group: %w", id, err)
			}
			child := map[string]any{observabilityGroupElement: groupChildren}
			layout, err := buildObservabilityLayoutOptions(container.Group.Layout)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("container %s group layout: %w", id, err)
			}
			if len(layout) > 0 {
				child["layout"] = layout
			}
			children[i] = child
			for key, value := range groupSaved {
				saved[key] = value
			}
			imports = append(imports, groupImports...)
		default:
			child, reference, ok := buildObservabilityPanelChild(spec, container.Template, id)
			children[i] = child
			if ok {
				imports = append(imports, reference)
			}
		}
		item, err := buildObservabilityLayoutItem(id, container.Layout)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("container %s layout: %w", id, err)
		}
		items[i] = item
	}

	entry := map[string]any{"items": items}
	for key, value := range metadata {
		entry[key] = value
	}
	saved[listKey] = entry
	return children, saved, imports, nil
}

// buildObservabilityPanelChild represents a reusable Template as a Panel child
// and adds the corresponding top-level Dashify import declaration.
func buildObservabilityPanelChild(spec map[string]any, ref *observabilityTemplateReferenceModel, id string) (map[string]any, string, bool) {
	if ref == nil {
		return nil, "", false
	}
	alias := observabilityImportAlias(id)
	reference := observabilityTemplatePrefix + ref.TemplateID.ValueString()
	spec[observabilityImportPrefix+alias] = reference
	return map[string]any{
		observabilityPanelElement: []any{
			map[string]any{observabilityImportElement + alias + ">": []any{}},
		},
	}, reference, true
}

// observabilityImportAlias derives a stable import name from the container's
// positional layout ID so the element and declaration can be paired on read.
func observabilityImportAlias(id string) string {
	return "widget" + strings.ReplaceAll(strings.TrimPrefix(id, "_."), ".", "_")
}

// parseDashboardTemplate consumes every JSON field represented by the model.
// Anything left over is important: Update rebuilds the complete document and
// would otherwise drop that content without telling the practitioner.
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
	if err := json.Unmarshal(record.Spec, &spec); err != nil || spec == nil {
		if err == nil {
			err = fmt.Errorf("spec is not a JSON object")
		}
		return model, formatDashboardParseError(err)
	}
	model.Title = types.StringValue(record.Title)
	if title, ok := spec["title"]; ok {
		if titleValue, ok := title.(string); !ok {
			return model, formatDashboardParseError(fmt.Errorf("spec.title is %T rather than a string", title))
		} else if titleValue != record.Title {
			return model, unsupportedDashboardSpec(fmt.Sprintf("record title %q conflicts with spec.title %q", record.Title, titleValue))
		}
	}
	delete(spec, "title")
	controlBar, controlLeftovers, err := parseObservabilityControlBar(spec)
	if err != nil {
		return model, formatDashboardParseError(err)
	}
	model.ControlBar = controlBar
	children, ok := spec[observabilityDashboardElement].([]any)
	if !ok {
		return model, unsupportedDashboardSpec(fmt.Sprintf("spec has no %q element list", observabilityDashboardElement))
	}
	delete(spec, observabilityDashboardElement)
	layout, err := parseObservabilityLayoutOptions(spec)
	if err != nil {
		return model, formatDashboardParseError(err)
	}
	model.Layout = layout

	used := map[string]bool{}
	containers, _, leftovers, err := parseObservabilityContainerList(spec, used, "_", "container", children, observabilityDashboardContainerLevel)
	if err != nil {
		return model, formatDashboardParseError(err)
	}
	model.Container = observabilityDashboardModelsFromContainers(containers)
	leftovers = append(controlLeftovers, leftovers...)
	for alias := range used {
		delete(spec, observabilityImportPrefix+alias)
	}
	leftovers = append(leftovers, observabilityLeftovers("", spec)...)
	if raw := strings.TrimSpace(string(record.SignalView)); raw != "" && raw != "null" {
		leftovers = append(leftovers, "signalview")
	}
	if record.Type != "" && record.Type != template.RecordType {
		leftovers = append(leftovers, "type")
	}
	if len(leftovers) > 0 {
		sort.Strings(leftovers)
		diags.AddWarning(
			"Dashboard fields not represented in Terraform state",
			fmt.Sprintf(
				"The stored dashboard contains %d field(s) this provider does not model:\n  %s\n\nThese fields are absent from Terraform state. A later update replaces the complete dashboard document and removes them.",
				len(leftovers),
				strings.Join(truncateObservabilityLeftovers(leftovers, observabilityLeftoverLimit), "\n  "),
			),
		)
	}
	return model, diags
}

// parseObservabilityContainerList decodes one container level and joins each
// content element with its separately stored positional layout entry.
func parseObservabilityContainerList(spec map[string]any, used map[string]bool, listKey, path string, children []any, level observabilityContainerLevel) ([]observabilityContainer, map[string]any, []string, error) {
	layouts, metadata, leftovers, err := parseObservabilityLayouts(spec, listKey, len(children))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("layout %q: %w", listKey, err)
	}
	containers := make([]observabilityContainer, len(children))
	for i, child := range children {
		id := fmt.Sprintf("%s.%d", listKey, i)
		containerPath := fmt.Sprintf("%s.%d", path, i)
		container, containerLeftovers, err := parseObservabilityContainer(spec, used, id, containerPath, child, level)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("container %s: %w", id, err)
		}
		container.Layout = layouts[i]
		containers[i] = container
		leftovers = append(leftovers, containerLeftovers...)
	}
	return containers, metadata, leftovers, nil
}

// parseObservabilityContainer dispatches one Dashify element according to the
// content types allowed at its dashboard, section, or group nesting level.
func parseObservabilityContainer(spec map[string]any, used map[string]bool, id, path string, raw any, level observabilityContainerLevel) (observabilityContainer, []string, error) {
	var container observabilityContainer
	node, ok := raw.(map[string]any)
	if !ok {
		return container, nil, fmt.Errorf("is %T rather than an element object", raw)
	}
	tag, value, err := oneObservabilityElement(node)
	if err != nil {
		return container, nil, err
	}
	switch tag {
	case observabilitySectionElement:
		if level != observabilityDashboardContainerLevel {
			return container, nil, observabilityUnexpectedContainerElement(tag, level)
		}
		items, ok := value.([]any)
		if !ok {
			return container, nil, fmt.Errorf("section value is %T rather than a list", value)
		}
		delete(node, tag)
		layout, err := parseObservabilityLayoutOptions(node)
		if err != nil {
			return container, nil, fmt.Errorf("section: %w", err)
		}
		sectionContainers, metadata, childLeftovers, err := parseObservabilityContainerList(spec, used, id, path+".section.container", items, observabilitySectionContainerLevel)
		if err != nil {
			return container, nil, fmt.Errorf("section: %w", err)
		}
		title, collapse, collapsible, err := parseObservabilitySectionMetadata(metadata)
		if err != nil {
			return container, nil, err
		}
		container.Section = &observabilitySection{Title: title, Collapse: collapse, Collapsible: collapsible, Layout: layout, Container: sectionContainers}
		return container, append(observabilityLeftovers(path, node), childLeftovers...), nil
	case observabilityGroupElement:
		if level == observabilityGroupContainerLevel {
			return container, nil, observabilityUnexpectedContainerElement(tag, level)
		}
		items, ok := value.([]any)
		if !ok {
			return container, nil, fmt.Errorf("group value is %T rather than a list", value)
		}
		delete(node, tag)
		layout, err := parseObservabilityLayoutOptions(node)
		if err != nil {
			return container, nil, fmt.Errorf("group: %w", err)
		}
		groupContainers, metadata, childLeftovers, err := parseObservabilityContainerList(spec, used, id, path+".group.container", items, observabilityGroupContainerLevel)
		if err != nil {
			return container, nil, fmt.Errorf("group: %w", err)
		}
		title, headerless, err := parseObservabilityGroupMetadata(metadata)
		if err != nil {
			return container, nil, err
		}
		container.Group = &observabilityGroup{Title: title, Headerless: headerless, Layout: layout, Container: groupContainers}
		return container, append(observabilityLeftovers(path, node), childLeftovers...), nil
	case observabilityPanelElement:
		ref, contentLeftovers, err := parseObservabilityPanel(spec, used, id, path, value)
		if err != nil {
			return container, nil, err
		}
		delete(node, tag)
		container.Template = ref
		return container, append(observabilityLeftovers(path, node), contentLeftovers...), nil
	default:
		return container, nil, observabilityUnexpectedContainerElement(tag, level)
	}
}

// observabilityUnexpectedContainerElement describes the legal children at a
// nesting level when a stored Dashify document cannot map to Terraform state.
func observabilityUnexpectedContainerElement(tag string, level observabilityContainerLevel) error {
	supported := "panels"
	switch level {
	case observabilitySectionContainerLevel:
		supported = "panels and groups"
	case observabilityDashboardContainerLevel:
		supported = "panels, sections, and groups"
	}
	return fmt.Errorf("is a %s element; only %s are supported at this level", tag, supported)
}

// parseObservabilityPanel resolves the Panel's import element to a Template ID.
// It also accepts the optional Chart wrapper emitted by some stored documents.
func parseObservabilityPanel(spec map[string]any, used map[string]bool, id, path string, raw any) (*observabilityTemplateReferenceModel, []string, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("panel value is %T rather than a list", raw)
	}
	if len(items) != 1 {
		return nil, nil, fmt.Errorf("panel contains %d elements; exactly one template reference is supported", len(items))
	}
	content, ok := items[0].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("panel content is %T rather than an element object", items[0])
	}
	tag, value, err := oneObservabilityElement(content)
	if err != nil {
		return nil, nil, err
	}
	elementContent := content
	wrapped := false
	if tag == "<Chart>" {
		chartItems, valid := value.([]any)
		if !valid || len(chartItems) != 1 {
			return nil, nil, fmt.Errorf("container %s contains an invalid Chart wrapper; exactly one chart element is required", id)
		}
		chartContent, valid := chartItems[0].(map[string]any)
		if !valid {
			return nil, nil, fmt.Errorf("container %s contains a Chart value of type %T", id, chartItems[0])
		}
		delete(content, tag)
		elementContent = chartContent
		wrapped = true
		tag, value, err = oneObservabilityElement(chartContent)
		if err != nil {
			return nil, nil, err
		}
	}
	if !strings.HasPrefix(tag, observabilityImportElement) {
		// TODO(charts): Dispatch the Dashify element tag through the generated chart
		// parser and preserve its unsupported-field/leftover checks.
		return nil, nil, fmt.Errorf("container %s contains unsupported inline element %s; typed inline charts are not supported by this resource; use an observability_template reference", id, tag)
	}
	alias := strings.TrimSuffix(strings.TrimPrefix(tag, observabilityImportElement), ">")
	reference, ok := spec[observabilityImportPrefix+alias].(string)
	if !ok {
		return nil, nil, fmt.Errorf("import %q has no matching declaration", alias)
	}
	idValue, ok := strings.CutPrefix(reference, observabilityTemplatePrefix)
	if !ok || idValue == "" {
		return nil, nil, fmt.Errorf("import %q references %q rather than a %s<id> template", alias, reference, observabilityTemplatePrefix)
	}
	used[alias] = true
	// Import elements normally carry an empty argument list. If they carry
	// anything else, leave the tag behind so Read warns that an update drops it.
	if args, ok := value.([]any); ok && len(args) == 0 {
		delete(elementContent, tag)
	}
	leftovers := observabilityLeftovers(path, content)
	if wrapped {
		leftovers = append(leftovers, observabilityLeftovers(path, elementContent)...)
	}
	return &observabilityTemplateReferenceModel{TemplateID: types.StringValue(idValue)}, leftovers, nil
}

// oneObservabilityElement finds the single angle-bracket Dashify element in an
// object while allowing ordinary sibling properties to be handled separately.
func oneObservabilityElement(node map[string]any) (string, any, error) {
	var tag string
	var value any
	for key, raw := range node {
		if strings.HasPrefix(key, "<") {
			if tag != "" {
				return "", nil, fmt.Errorf("has multiple Dashify element keys")
			}
			tag = key
			value = raw
		}
	}
	if tag == "" {
		return "", nil, fmt.Errorf("has no Dashify element key")
	}
	return tag, value, nil
}

// observabilityLeftovers returns the leaf paths not consumed by the typed
// parser. Arrays are reported as a unit because Terraform does not model any
// part of an unknown array.
func observabilityLeftovers(prefix string, node map[string]any) []string {
	var leftovers []string
	for key, value := range node {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		switch value := value.(type) {
		case nil:
		case map[string]any:
			leftovers = append(leftovers, observabilityLeftovers(path, value)...)
		case []any:
			if len(value) > 0 {
				leftovers = append(leftovers, path)
			}
		default:
			leftovers = append(leftovers, path)
		}
	}
	return leftovers
}

// truncateObservabilityLeftovers bounds warning output while retaining the
// number of additional unmodeled paths that were omitted from the message.
func truncateObservabilityLeftovers(leftovers []string, limit int) []string {
	if len(leftovers) <= limit {
		return leftovers
	}
	return append(leftovers[:limit:limit], fmt.Sprintf("... and %d more", len(leftovers)-limit))
}
