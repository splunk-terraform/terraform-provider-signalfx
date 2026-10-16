// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package interpreter loads curated YAML chart definitions and interprets
// them as Terraform attributes, validation rules, and JSON mappings.
package interpreter

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type Definition struct {
	Type        string              `yaml:"type"`
	Version     int                 `yaml:"version"`
	Description string              `yaml:"description"`
	Element     string              `yaml:"element"`
	Extends     []string            `yaml:"extends"`
	Properties  map[string]Property `yaml:"properties"`
}

type Mapping struct {
	Path      []string `yaml:"path"`
	Transform string   `yaml:"transform"`
}
type Item struct {
	Type       string              `yaml:"type"`
	Properties map[string]Property `yaml:"properties"`
}
type Condition struct {
	Field string `yaml:"field"`
	Value string `yaml:"value"`
}
type Property struct {
	Type          string              `yaml:"type"`
	Description   string              `yaml:"description"`
	Ref           string              `yaml:"ref"`
	Optional      bool                `yaml:"optional"`
	Required      bool                `yaml:"required"`
	Values        []string            `yaml:"values"`
	Default       any                 `yaml:"default"`
	Min           *int                `yaml:"min"`
	Max           *int                `yaml:"max"`
	MinLength     *int                `yaml:"min_length"`
	MaxItems      *int                `yaml:"max_items"`
	ConflictsWith []string            `yaml:"conflicts_with"`
	AtLeastOneOf  []string            `yaml:"at_least_one_of"`
	RequiresValue *Condition          `yaml:"requires_value"`
	RequiredWhen  *Condition          `yaml:"required_when"`
	Mapping       Mapping             `yaml:"mapping"`
	Item          *Item               `yaml:"item"`
	Properties    map[string]Property `yaml:"properties"`
	Variants      map[string]Property `yaml:"variants"`
}

// Definitions and public schema views must not share mutable authored data
// with the cached engine, which serves concurrent resource operations.
func copyDefinition(d Definition) Definition {
	d.Extends = slices.Clone(d.Extends)
	d.Properties = copyProperties(d.Properties)
	return d
}

func copyProperties(properties map[string]Property) map[string]Property {
	if properties == nil {
		return nil
	}
	result := make(map[string]Property, len(properties))
	for name, p := range properties {
		p.Values, p.Mapping.Path = slices.Clone(p.Values), slices.Clone(p.Mapping.Path)
		p.ConflictsWith, p.AtLeastOneOf = slices.Clone(p.ConflictsWith), slices.Clone(p.AtLeastOneOf)
		p.Min, p.Max, p.MinLength, p.MaxItems = copyPointer(p.Min), copyPointer(p.Max), copyPointer(p.MinLength), copyPointer(p.MaxItems)
		p.RequiresValue, p.RequiredWhen = copyPointer(p.RequiresValue), copyPointer(p.RequiredWhen)
		p.Default = clone(p.Default)
		p.Properties, p.Variants = copyProperties(p.Properties), copyProperties(p.Variants)
		if p.Item != nil {
			p.Item = &Item{Type: p.Item.Type, Properties: copyProperties(p.Item.Properties)}
		}
		result[name] = p
	}
	return result
}

func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// Reject declarations the interpreter cannot enforce rather than silently
// accepting a newer dialect or producing ambiguous wire mappings.
func checkProperties(properties map[string]Property) error {
	names := keys(properties)
	for index, name := range names {
		p := properties[name]
		fail := func(message string) error { return fmt.Errorf("%s: %s", name, message) }
		if p.Values != nil && p.Type != "enum" {
			return fail("values require enum")
		}
		if (p.Min != nil || p.Max != nil) && p.Type != "number" {
			return fail("numeric bounds require number")
		}
		if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
			return fail("minimum exceeds maximum")
		}
		if p.MinLength != nil && ((*p.MinLength < 0) || (p.Type != "string" && p.Type != "enum")) {
			return fail("min_length needs a string and a nonnegative value")
		}
		if p.MaxItems != nil && (p.Type != "list" || *p.MaxItems < 0) {
			return fail("max_items needs a list and a nonnegative value")
		}
		if p.Item != nil && p.Type != "list" && p.Type != "map" {
			return fail("item requires list or map")
		}
		if len(p.Properties) > 0 && p.Type != "block" {
			return fail("properties require block")
		}
		if len(p.Variants) > 0 && p.Type != "oneof" {
			return fail("variants require oneof")
		}
		if len(p.AtLeastOneOf) > 0 && p.Type != "block" && p.Type != "oneof" {
			return fail("at_least_one_of requires object")
		}
		if p.Type == "block" && len(p.Properties) == 0 {
			return fail("block has no properties")
		}
		if (p.Type == "list" || p.Type == "map") && p.Item != nil && p.Item.Type == "" && len(p.Item.Properties) == 0 {
			return fail("object item has no properties")
		}
		if p.Item != nil && p.Item.Type != "" && len(p.Item.Properties) > 0 {
			return fail("item must select scalar type or object properties")
		}
		for _, peer := range p.ConflictsWith {
			if peer == name || properties[peer].Type == "" {
				return fail("unknown or self conflict: " + peer)
			}
		}
		for _, condition := range []*Condition{p.RequiresValue, p.RequiredWhen} {
			if condition == nil {
				continue
			}
			peer, exists := properties[condition.Field]
			if !exists || condition.Field == name || (peer.Type != "string" && peer.Type != "enum") {
				return fail("condition needs a sibling string field")
			}
			if peer.Type == "enum" && !slices.Contains(peer.Values, condition.Value) {
				return fail("condition uses value outside sibling enum")
			}
		}
		if len(p.AtLeastOneOf) > 0 {
			children, err := childProperties(p)
			if err != nil {
				return err
			}
			for _, field := range p.AtLeastOneOf {
				if children[field].Type == "" {
					return fail("unknown at_least_one_of field: " + field)
				}
			}
		}
		for _, otherName := range names[index+1:] {
			other := properties[otherName]
			left, right := strings.Join(p.Mapping.Path, "\x00"), strings.Join(other.Mapping.Path, "\x00")
			if left != "" && right != "" && (left == right || strings.HasPrefix(left, right+"\x00") || strings.HasPrefix(right, left+"\x00")) {
				return fail("mapping overlaps " + otherName)
			}
		}
	}
	return nil
}

// Block variants contribute fields to the parent object; they do not retain a
// separate value on which presence, mappings, or constraints could operate.
func checkBlockVariant(p Property) error {
	allowed := Property{Type: p.Type, Description: p.Description, Properties: p.Properties}
	if !reflect.DeepEqual(p, allowed) {
		return fmt.Errorf("oneof block variants support only type, description, and properties; put constraints on the parent or its fields")
	}
	if len(p.Properties) == 0 {
		return fmt.Errorf("oneof block variant has no properties")
	}
	return nil
}
