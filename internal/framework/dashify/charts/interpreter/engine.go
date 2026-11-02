// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Chart struct {
	Name string
	// Element is the exact serialized tag, including angle brackets.
	Element     string
	Description string
}

func attributeTypes(attributes map[string]schema.Attribute) map[string]attr.Type {
	result := make(map[string]attr.Type, len(attributes))
	for name, attribute := range attributes {
		result[name] = attribute.GetType()
	}
	return result
}

// Mappings with an intermediate root declare chart container objects. Seeding
// only those roots keeps Text independent of the metrics datasource container.
func containerRoots(properties map[string]property) []string {
	roots := map[string]bool{}
	for _, property := range properties {
		if len(property.Mapping.Path) > 1 {
			roots[property.Mapping.Path[0]] = true
		}
	}
	return keys(roots)
}

type Engine struct {
	definitions map[string]definition
	attributes  map[string]map[string]schema.Attribute
	roots       map[string][]string
}

// newEngine takes ownership of definitions decoded by load.
func newEngine(definitions []definition) (*Engine, error) {
	e := &Engine{definitions: map[string]definition{}, attributes: map[string]map[string]schema.Attribute{}, roots: map[string][]string{}}
	for _, definition := range definitions {
		if definition.Type == "id" || definition.Type == "title" {
			return nil, fmt.Errorf("chart name %q is reserved by the resource", definition.Type)
		}
		if definition.Type == "" || definition.Element == "" {
			return nil, fmt.Errorf("chart %q needs a type and an element name", definition.Type)
		}
		if _, exists := e.definitions[definition.Type]; exists {
			return nil, fmt.Errorf("duplicate chart %q", definition.Type)
		}
		for _, other := range e.definitions {
			if other.Element == definition.Element {
				return nil, fmt.Errorf("duplicate element %q", definition.Element)
			}
		}
		attributes, err := buildAttributes(definition.Properties)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", definition.Type, err)
		}
		e.definitions[definition.Type], e.attributes[definition.Type] = definition, attributes
		e.roots[definition.Type] = containerRoots(definition.Properties)
	}
	return e, nil
}

func (e *Engine) Charts() []Chart {
	result := make([]Chart, 0, len(e.definitions))
	for _, name := range keys(e.definitions) {
		d := e.definitions[name]
		result = append(result, Chart{Name: name, Element: "<" + d.Element + ">", Description: d.Description})
	}
	return result
}

// Attributes returns the cached chart schema. Callers must not modify it.
func (e *Engine) Attributes(name string) (map[string]schema.Attribute, error) {
	attributes, exists := e.attributes[name]
	if !exists {
		return nil, fmt.Errorf("unknown chart %q", name)
	}
	return attributes, nil
}

func buildAttributes(properties map[string]property) (map[string]schema.Attribute, error) {
	if err := checkProperties(properties); err != nil {
		return nil, err
	}
	attributes := make(map[string]schema.Attribute, len(properties))
	for _, name := range keys(properties) {
		p := properties[name]
		if p.Ref != "" || p.Description == "" || len(p.Mapping.Path) == 0 || p.Required == p.Optional {
			return nil, fmt.Errorf("%s: resolve refs and supply description, mapping, and one presence mode", name)
		}
		for _, segment := range p.Mapping.Path {
			if segment == "" {
				return nil, fmt.Errorf("%s: empty mapping segment", name)
			}
		}
		if p.Mapping.Transform != "" && p.Mapping.Transform != transformNegate && p.Mapping.Transform != transformRelativeDuration {
			return nil, fmt.Errorf("%s: unsupported transform %q", name, p.Mapping.Transform)
		}
		if (p.Mapping.Transform == transformNegate && p.Type != propertyTypeBool) || (p.Mapping.Transform == transformRelativeDuration && p.Type != propertyTypeString) {
			return nil, fmt.Errorf("%s: transform does not match type", name)
		}
		description := p.Description
		if p.Default != nil {
			description += fmt.Sprintf(" Defaults to `%v` when unset.", p.Default)
		}
		//exhaustive:enforce
		switch p.Type {
		case propertyTypeString, propertyTypeEnum:
			if p.Type == propertyTypeEnum {
				if len(p.Values) == 0 {
					return nil, fmt.Errorf("%s: enum has no values", name)
				}
			}
			attributes[name] = schema.StringAttribute{Required: p.Required, Optional: !p.Required, MarkdownDescription: description}
		case propertyTypeBool:
			attributes[name] = schema.BoolAttribute{Required: p.Required, Optional: !p.Required, MarkdownDescription: description}
		case propertyTypeNumber:
			attributes[name] = schema.Int64Attribute{Required: p.Required, Optional: !p.Required, MarkdownDescription: description}
		case propertyTypeFloat:
			attributes[name] = schema.Float64Attribute{Required: p.Required, Optional: !p.Required, MarkdownDescription: description}
		case propertyTypeBlock, propertyTypeOneOf, propertyTypeList, propertyTypeMap:
			children, err := childProperties(p)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if p.Type == propertyTypeList && p.Item != nil && p.Item.Type == propertyTypeString {
				attributes[name] = schema.ListAttribute{Required: p.Required, Optional: !p.Required, ElementType: types.StringType, MarkdownDescription: description}
				break
			}
			nested, err := buildAttributes(children)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			object := schema.NestedAttributeObject{Attributes: nested}
			switch p.Type {
			case propertyTypeBlock, propertyTypeOneOf:
				attributes[name] = schema.SingleNestedAttribute{Required: p.Required, Optional: !p.Required, Attributes: nested, MarkdownDescription: description}
			case propertyTypeList:
				attributes[name] = schema.ListNestedAttribute{Required: p.Required, Optional: !p.Required, NestedObject: object, MarkdownDescription: description}
			case propertyTypeMap:
				attributes[name] = schema.MapNestedAttribute{Required: p.Required, Optional: !p.Required, NestedObject: object, MarkdownDescription: description}
			}
		default:
			return nil, fmt.Errorf("%s: unsupported type %q", name, p.Type)
		}
	}
	return attributes, nil
}

// Oneof variants share one Terraform object: scalar variants become named
// fields, while block variants contribute their children. Wire shape selects
// the active variant; mutually exclusive variants cannot be silently combined.
func childProperties(p property) (map[string]property, error) {
	switch p.Type {
	case propertyTypeBlock:
		return p.Properties, nil
	case propertyTypeList, propertyTypeMap:
		if p.Item == nil {
			return nil, fmt.Errorf("collection requires item")
		}
		if p.Item.Type != "" && (p.Item.Type != propertyTypeString || p.Type != propertyTypeList) {
			return nil, fmt.Errorf("unsupported %s item type %q", p.Type, p.Item.Type)
		}
		return p.Item.Properties, nil
	case propertyTypeOneOf:
		out := map[string]property{}
		scalarVariants := 0
		for variant, definition := range p.Variants {
			children := definition.Properties
			if definition.Type == propertyTypeBlock {
				if err := checkBlockVariant(definition); err != nil {
					return nil, fmt.Errorf("%s: %w", variant, err)
				}
			} else {
				if definition.Type != propertyTypeString && definition.Type != propertyTypeEnum {
					return nil, fmt.Errorf("unsupported oneof variant type %q", definition.Type)
				}
				scalarVariants++
				if scalarVariants > 1 {
					return nil, fmt.Errorf("oneof has ambiguous scalar variants")
				}
				definition.Mapping.Path = []string{variant}
				definition.Required, definition.Optional = false, true
				children = map[string]property{variant: definition}
			}
			for name, child := range children {
				if definition.Type == propertyTypeBlock && child.Required {
					return nil, fmt.Errorf("oneof block field %q cannot be required", name)
				}
				if _, exists := out[name]; exists {
					return nil, fmt.Errorf("overlapping oneof field %q", name)
				}
				out[name] = child
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("oneof requires variants")
		}
		return out, nil
	default:
		return nil, nil
	}
}

func (e *Engine) Validate(name string, value types.Object) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	d, exists := e.definitions[name]
	if !exists {
		diagnostics.AddError("Invalid chart", fmt.Sprintf("Unknown chart %q.", name))
		return diagnostics
	}
	validateValue(name, property{Type: propertyTypeBlock, Required: true, Properties: d.Properties}, value, &diagnostics)
	return diagnostics
}

func validateValue(path string, p property, value attr.Value, diagnostics *diag.Diagnostics) {
	fail := func(message string) { diagnostics.AddError("Invalid chart value", path+": "+message) }
	if value == nil || value.IsNull() {
		if p.Required {
			fail("must be set")
		}
		return
	}
	if value.IsUnknown() {
		return
	}
	switch v := value.(type) {
	case types.String:
		if p.MinLength != nil && utf8.RuneCountInString(v.ValueString()) < *p.MinLength {
			fail(fmt.Sprintf("must contain at least %d characters", *p.MinLength))
		}
		if p.Type == propertyTypeEnum && !slices.Contains(p.Values, v.ValueString()) {
			fail(fmt.Sprintf("must be one of %v", p.Values))
		}
		if p.Mapping.Transform == transformRelativeDuration {
			if _, valid := relativeDurationSpec(v.ValueString()); !valid {
				fail("must be a nonzero relative time range such as -15m or -1h30m, representable as whole seconds")
			}
		}
	case types.Int64:
		if p.Min != nil && v.ValueInt64() < int64(*p.Min) {
			fail(fmt.Sprintf("must be at least %d", *p.Min))
		}
		if p.Max != nil && v.ValueInt64() > int64(*p.Max) {
			fail(fmt.Sprintf("must be at most %d", *p.Max))
		}
	case types.Float64:
		if math.IsNaN(v.ValueFloat64()) || math.IsInf(v.ValueFloat64(), 0) {
			fail("must be a finite number")
		}
	case types.Object:
		children, _ := childProperties(p)
		attributes := v.Attributes()
		for _, name := range keys(children) {
			child := children[name]
			validateValue(join(path, name), child, attributes[name], diagnostics)
			if present(attributes[name]) {
				for _, other := range child.ConflictsWith {
					if present(attributes[other]) {
						fail(name + " conflicts with " + other)
					}
				}
				if condition := child.RequiresValue; condition != nil && !matches(attributes[condition.Field], condition.Value) && !unknown(attributes[condition.Field]) {
					fail(name + " requires " + condition.Field + " = " + condition.Value)
				}
			}
			if condition := child.RequiredWhen; condition != nil && !unknown(attributes[name]) && !nonempty(attributes[name]) && matches(attributes[condition.Field], condition.Value) {
				fail(name + " must be set when " + condition.Field + " = " + condition.Value)
			}
		}
		if p.Type == propertyTypeOneOf {
			active := 0
			for name, variant := range p.Variants {
				fields := []string{name}
				if variant.Type == propertyTypeBlock {
					fields = keys(variant.Properties)
				}
				for _, field := range fields {
					if present(attributes[field]) {
						active++
						break
					}
				}
			}
			if active > 1 {
				fail("oneof variants conflict")
			}
		}
		if len(p.AtLeastOneOf) > 0 {
			found := false
			for _, name := range p.AtLeastOneOf {
				if nonempty(attributes[name]) || unknown(attributes[name]) {
					found = true
				}
			}
			if !found {
				fail(fmt.Sprintf("must set at least one of %v", p.AtLeastOneOf))
			}
		}
	case types.List:
		if p.MaxItems != nil && len(v.Elements()) > *p.MaxItems {
			fail(fmt.Sprintf("must contain at most %d items", *p.MaxItems))
		}
		item := property{Type: propertyTypeBlock, Required: true}
		if p.Item != nil {
			item.Properties = p.Item.Properties
			if p.Item.Type == propertyTypeString {
				item.Type = propertyTypeString
			}
		}
		for index, child := range v.Elements() {
			validateValue(fmt.Sprintf("%s.%d", path, index), item, child, diagnostics)
		}
	case types.Map:
		item := property{Type: propertyTypeBlock, Required: true}
		if p.Item != nil {
			item.Properties = p.Item.Properties
		}
		for _, key := range keys(v.Elements()) {
			validateValue(join(path, key), item, v.Elements()[key], diagnostics)
		}
	}
}

func keys[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
func join(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
func unknown(v attr.Value) bool { return v != nil && v.IsUnknown() }
func present(v attr.Value) bool { return v != nil && !v.IsNull() && !v.IsUnknown() }
func nonempty(v attr.Value) bool {
	if !present(v) {
		return false
	}
	if s, ok := v.(types.String); ok {
		return s.ValueString() != ""
	}
	if list, ok := v.(types.List); ok {
		return len(list.Elements()) > 0
	}
	if items, ok := v.(types.Map); ok {
		return len(items.Elements()) > 0
	}
	return true
}
func matches(v attr.Value, expected string) bool {
	value, ok := v.(types.String)
	return ok && present(value) && value.ValueString() == expected
}
