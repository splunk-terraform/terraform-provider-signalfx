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
		//exhaustive:enforce
		switch p.Type {
		case propertyTypeString:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMaxItems, checkNoItem, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
			if p.MinLength != nil && *p.MinLength < 0 {
				return fmt.Errorf("%s: min_length needs a string and a nonnegative value", name)
			}
		case propertyTypeEnum:
			if err := checkPropertyFields(name, p, checkNoBounds, checkNoMaxItems, checkNoItem, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
			if p.MinLength != nil && *p.MinLength < 0 {
				return fmt.Errorf("%s: min_length needs a string and a nonnegative value", name)
			}
		case propertyTypeNumber:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoMinLength, checkNoMaxItems, checkNoItem, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
			if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
				return fmt.Errorf("%s: minimum exceeds maximum", name)
			}
		case propertyTypeBool, propertyTypeFloat:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMinLength, checkNoMaxItems, checkNoItem, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
		case propertyTypeList:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMinLength, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
			if p.MaxItems != nil && *p.MaxItems < 0 {
				return fmt.Errorf("%s: max_items needs a list and a nonnegative value", name)
			}
			if err := checkItem(name, p); err != nil {
				return err
			}
		case propertyTypeMap:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMinLength, checkNoMaxItems, checkNoProperties, checkNoVariants, checkNoAtLeastOneOf); err != nil {
				return err
			}
			if err := checkItem(name, p); err != nil {
				return err
			}
		case propertyTypeBlock:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMinLength, checkNoMaxItems, checkNoItem, checkNoVariants); err != nil {
				return err
			}
			if len(p.Properties) == 0 {
				return fmt.Errorf("%s: block has no properties", name)
			}
			if err := checkAtLeastOneOf(name, p); err != nil {
				return err
			}
		case propertyTypeOneOf:
			if err := checkPropertyFields(name, p, checkNoValues, checkNoBounds, checkNoMinLength, checkNoMaxItems, checkNoItem, checkNoProperties); err != nil {
				return err
			}
			if err := checkAtLeastOneOf(name, p); err != nil {
				return err
			}
		}
		for _, peer := range p.ConflictsWith {
			if peer == name || properties[peer].Type == "" {
				return fmt.Errorf("%s: unknown or self conflict: %s", name, peer)
			}
		}
		for _, condition := range []*condition{p.RequiresValue, p.RequiredWhen} {
			if condition == nil {
				continue
			}
			peer, exists := properties[condition.Field]
			if !exists || condition.Field == name || (peer.Type != propertyTypeString && peer.Type != propertyTypeEnum) {
				return fmt.Errorf("%s: condition needs a sibling string field", name)
			}
			if peer.Type == propertyTypeEnum && !slices.Contains(peer.Values, condition.Value) {
				return fmt.Errorf("%s: condition uses value outside sibling enum", name)
			}
		}
		for _, otherName := range names[index+1:] {
			other := properties[otherName]
			left, right := strings.Join(p.Mapping.Path, "\x00"), strings.Join(other.Mapping.Path, "\x00")
			if left != "" && right != "" && (left == right || strings.HasPrefix(left, right+"\x00") || strings.HasPrefix(right, left+"\x00")) {
				return fmt.Errorf("%s: mapping overlaps %s", name, otherName)
			}
		}
	}
	return nil
}

// checkPropertyFields runs the selected checks in order and returns the first error.
func checkPropertyFields(name string, p property, checks ...func(string, property) error) error {
	for _, check := range checks {
		if err := check(name, p); err != nil {
			return err
		}
	}
	return nil
}

func checkNoValues(name string, p property) error {
	if p.Values != nil {
		return fmt.Errorf("%s: values require enum", name)
	}
	return nil
}

func checkNoBounds(name string, p property) error {
	if p.Min != nil || p.Max != nil {
		return fmt.Errorf("%s: numeric bounds require number", name)
	}
	return nil
}

func checkNoMinLength(name string, p property) error {
	if p.MinLength != nil {
		return fmt.Errorf("%s: min_length needs a string and a nonnegative value", name)
	}
	return nil
}

func checkNoMaxItems(name string, p property) error {
	if p.MaxItems != nil {
		return fmt.Errorf("%s: max_items needs a list and a nonnegative value", name)
	}
	return nil
}

func checkNoItem(name string, p property) error {
	if p.Item != nil {
		return fmt.Errorf("%s: item requires list or map", name)
	}
	return nil
}

func checkNoProperties(name string, p property) error {
	if len(p.Properties) > 0 {
		return fmt.Errorf("%s: properties require block", name)
	}
	return nil
}

func checkNoVariants(name string, p property) error {
	if len(p.Variants) > 0 {
		return fmt.Errorf("%s: variants require oneof", name)
	}
	return nil
}

func checkNoAtLeastOneOf(name string, p property) error {
	if len(p.AtLeastOneOf) > 0 {
		return fmt.Errorf("%s: at_least_one_of requires object", name)
	}
	return nil
}

func checkItem(name string, p property) error {
	if p.Item == nil {
		return nil
	}
	if p.Item.Type == "" && len(p.Item.Properties) == 0 {
		return fmt.Errorf("%s: object item has no properties", name)
	}
	if p.Item.Type != "" && len(p.Item.Properties) > 0 {
		return fmt.Errorf("%s: item must select scalar type or object properties", name)
	}
	return nil
}

func checkAtLeastOneOf(name string, p property) error {
	if len(p.AtLeastOneOf) == 0 {
		return nil
	}
	children, err := childProperties(p)
	if err != nil {
		return err
	}
	for _, field := range p.AtLeastOneOf {
		if children[field].Type == "" {
			return fmt.Errorf("%s: unknown at_least_one_of field: %s", name, field)
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
