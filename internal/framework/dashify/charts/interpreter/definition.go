// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package interpreter loads curated YAML chart definitions and interprets
// them as Terraform attributes, validation rules, and JSON mappings.
package interpreter

import (
	"fmt"
	"slices"
	"strings"
)

type propertyType string

const (
	propertyTypeString propertyType = "string"
	propertyTypeEnum   propertyType = "enum"
	propertyTypeBool   propertyType = "bool"
	propertyTypeNumber propertyType = "number"
	propertyTypeFloat  propertyType = "float"
	propertyTypeBlock  propertyType = "block"
	propertyTypeOneOf  propertyType = "oneof"
	propertyTypeList   propertyType = "list"
	propertyTypeMap    propertyType = "map"
)

const (
	transformNegate           = "negate"
	transformRelativeDuration = "relative_duration"
)

type definition struct {
	Type        string              `yaml:"type"`
	Version     int                 `yaml:"version"`
	Description string              `yaml:"description"`
	Element     string              `yaml:"element"`
	Extends     []string            `yaml:"extends"`
	Properties  map[string]property `yaml:"properties"`
}

type mapping struct {
	Path      []string `yaml:"path"`
	Transform string   `yaml:"transform"`
}
type item struct {
	Type       propertyType        `yaml:"type"`
	Properties map[string]property `yaml:"properties"`
}
type condition struct {
	Field string `yaml:"field"`
	Value string `yaml:"value"`
}
type property struct {
	Type          propertyType        `yaml:"type"`
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
	RequiresValue *condition          `yaml:"requires_value"`
	RequiredWhen  *condition          `yaml:"required_when"`
	Mapping       mapping             `yaml:"mapping"`
	Item          *item               `yaml:"item"`
	Properties    map[string]property `yaml:"properties"`
	Variants      map[string]property `yaml:"variants"`
}

// Reject declarations the interpreter cannot enforce rather than silently
// accepting a newer dialect or producing ambiguous wire mappings.
func checkProperties(properties map[string]property) error {
	names := keys(properties)
	for index, name := range names {
		p := properties[name]
		fail := func(message string) error { return fmt.Errorf("%s: %s", name, message) }
		if p.Values != nil && p.Type != propertyTypeEnum {
			return fail("values require enum")
		}
		if (p.Min != nil || p.Max != nil) && p.Type != propertyTypeNumber {
			return fail("numeric bounds require number")
		}
		if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
			return fail("minimum exceeds maximum")
		}
		if p.MinLength != nil && ((*p.MinLength < 0) || (p.Type != propertyTypeString && p.Type != propertyTypeEnum)) {
			return fail("min_length needs a string and a nonnegative value")
		}
		if p.MaxItems != nil && (p.Type != propertyTypeList || *p.MaxItems < 0) {
			return fail("max_items needs a list and a nonnegative value")
		}
		if p.Item != nil && p.Type != propertyTypeList && p.Type != propertyTypeMap {
			return fail("item requires list or map")
		}
		if len(p.Properties) > 0 && p.Type != propertyTypeBlock {
			return fail("properties require block")
		}
		if len(p.Variants) > 0 && p.Type != propertyTypeOneOf {
			return fail("variants require oneof")
		}
		if len(p.AtLeastOneOf) > 0 && p.Type != propertyTypeBlock && p.Type != propertyTypeOneOf {
			return fail("at_least_one_of requires object")
		}
		if p.Type == propertyTypeBlock && len(p.Properties) == 0 {
			return fail("block has no properties")
		}
		if (p.Type == propertyTypeList || p.Type == propertyTypeMap) && p.Item != nil && p.Item.Type == "" && len(p.Item.Properties) == 0 {
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
		for _, condition := range []*condition{p.RequiresValue, p.RequiredWhen} {
			if condition == nil {
				continue
			}
			peer, exists := properties[condition.Field]
			if !exists || condition.Field == name || (peer.Type != propertyTypeString && peer.Type != propertyTypeEnum) {
				return fail("condition needs a sibling string field")
			}
			if peer.Type == propertyTypeEnum && !slices.Contains(peer.Values, condition.Value) {
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
func checkBlockVariant(p property) error {
	if p.Ref != "" || p.Optional || p.Required || p.Values != nil || p.Default != nil ||
		p.Min != nil || p.Max != nil || p.MinLength != nil || p.MaxItems != nil ||
		p.ConflictsWith != nil || p.AtLeastOneOf != nil || p.RequiresValue != nil || p.RequiredWhen != nil ||
		p.Mapping.Path != nil || p.Mapping.Transform != "" || p.Item != nil || p.Variants != nil {
		return fmt.Errorf("oneof block variants support only type, description, and properties; put constraints on the parent or its fields")
	}
	if len(p.Properties) == 0 {
		return fmt.Errorf("oneof block variant has no properties")
	}
	return nil
}
