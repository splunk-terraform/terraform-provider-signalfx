// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

// ValidateCharts checks actual encoded configurations against product schemas.
// Samples come only from curated declarations, never product-required fields,
// so stricter product schemas cannot conceal missing Terraform fields.
func ValidateCharts(ctx context.Context, engine *interpreter.Engine, catalog *Catalog) error {
	for _, chart := range engine.Charts() {
		attributes, err := engine.Attributes(chart.Name)
		if err != nil {
			return err
		}
		if diagnostics := (schema.Schema{Attributes: attributes}).ValidateImplementation(ctx); diagnostics.HasError() {
			return fmt.Errorf("%s: invalid Terraform schema: %v", chart.Name, diagnostics)
		}
		definition, _ := engine.Definition(chart.Name)
		cases, err := objectCases(ctx, definition.Properties, attributes, nil)
		if err != nil {
			return fmt.Errorf("%s: cannot construct curated configurations: %w", chart.Name, err)
		}
		for _, candidate := range cases {
			spec, diagnostics := engine.Encode(ctx, chart.Name, candidate.value.(types.Object))
			if diagnostics.HasError() {
				return fmt.Errorf("%s (%s): invalid curated configuration: %v", chart.Name, candidate.name, diagnostics)
			}
			if err := catalog.ValidateElement(chart.Element, spec); err != nil {
				return fmt.Errorf("%s (%s): actual encoded configuration violates product contract: %w", chart.Name, candidate.name, err)
			}
		}
	}
	return nil
}

type example struct {
	name  string
	value attr.Value
}

func names[T any](values map[string]T) []string { return slices.Sorted(maps.Keys(values)) }

// Activate each property independently, retaining required siblings. Recursive
// cases cover optional objects, representative list/map entries, every enum and
// numeric bounds without constructing a Cartesian product.
func objectCases(ctx context.Context, properties map[string]interpreter.Property, attributes map[string]schema.Attribute, needed []string) ([]example, error) {
	base := make(map[string]attr.Value, len(attributes))
	variants := map[string][]example{}
	for _, name := range names(attributes) {
		attribute, property := attributes[name], properties[name]
		cases, err := valueCases(ctx, property, attribute)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		variants[name] = cases
		value, err := nullValue(ctx, attribute)
		if err != nil {
			return nil, err
		}
		if property.Required {
			value = cases[0].value
		}
		base[name] = value
	}
	if len(needed) > 0 {
		found := false
		for _, name := range needed {
			found = found || nonempty(base[name])
		}
		if !found {
			value, err := firstNonempty(variants[needed[0]])
			if err != nil {
				return nil, err
			}
			base[needed[0]] = value
		}
	}
	if err := conditions(ctx, properties, attributes, variants, base, ""); err != nil {
		return nil, err
	}
	makeCase := func(name string, values map[string]attr.Value) example {
		return example{name, types.ObjectValueMust(interpreter.AttributeTypes(attributes), values)}
	}
	result := []example{makeCase("minimum", base)}
	for _, name := range names(attributes) {
		for _, candidate := range variants[name] {
			values := maps.Clone(base)
			values[name] = candidate.value
			if err := conditions(ctx, properties, attributes, variants, values, name); err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			result = append(result, makeCase(name+"."+candidate.name, values))
		}
	}
	return result, nil
}

func valueCases(ctx context.Context, p interpreter.Property, attribute schema.Attribute) ([]example, error) {
	scalar := func(value attr.Value) []example { return []example{{"set", value}} }
	switch p.Type {
	case "string", "enum":
		if p.Type == "enum" {
			result := make([]example, 0, len(p.Values))
			for _, value := range p.Values {
				result = append(result, example{value, types.StringValue(value)})
			}
			return result, nil
		}
		value := "value"
		if p.MinLength != nil && *p.MinLength > 0 {
			value = strings.Repeat("x", *p.MinLength)
		}
		if p.Mapping.Transform == "relative_duration" {
			value = "1h"
		}
		return scalar(types.StringValue(value)), nil
	case "bool":
		return []example{{"false", types.BoolValue(false)}, {"true", types.BoolValue(true)}}, nil
	case "number":
		minimum := 0
		if p.Min != nil {
			minimum = *p.Min
		} else if p.Max != nil && *p.Max < 0 {
			minimum = *p.Max
		}
		maximum := minimum
		if p.Max != nil {
			maximum = *p.Max
		}
		return []example{{"minimum", types.Int64Value(int64(minimum))}, {"maximum", types.Int64Value(int64(maximum))}}, nil
	case "float":
		return scalar(types.Float64Value(0.125)), nil
	case "block":
		return objectCases(ctx, p.Properties, attribute.(schema.SingleNestedAttribute).Attributes, p.AtLeastOneOf)
	case "oneof":
		all := attribute.(schema.SingleNestedAttribute).Attributes
		var result []example
		for _, name := range names(p.Variants) {
			variant := p.Variants[name]
			children := variant.Properties
			if variant.Type != "block" {
				variant.Required, variant.Optional = true, false
				children = map[string]interpreter.Property{name: variant}
			}
			subset := map[string]schema.Attribute{}
			for child := range children {
				subset[child] = all[child]
			}
			childNames := names(children)
			if len(childNames) == 0 {
				return nil, fmt.Errorf("oneof variant %s has no representable fields", name)
			}
			cases, err := objectCases(ctx, children, subset, childNames[:1])
			if err != nil {
				return nil, err
			}
			for _, candidate := range cases {
				values := map[string]attr.Value{}
				for field, fieldAttribute := range all {
					values[field], err = nullValue(ctx, fieldAttribute)
					if err != nil {
						return nil, err
					}
				}
				maps.Copy(values, candidate.value.(types.Object).Attributes())
				result = append(result, example{name + "." + candidate.name, types.ObjectValueMust(interpreter.AttributeTypes(all), values)})
			}
		}
		return result, nil
	case "list", "map":
		var entries []example
		var elementType attr.Type
		if p.Type == "list" && p.Item.Type == "string" {
			elementType, entries = types.StringType, scalar(types.StringValue("value"))
		} else {
			var nested map[string]schema.Attribute
			if p.Type == "list" {
				nested = attribute.(schema.ListNestedAttribute).NestedObject.Attributes
			} else {
				nested = attribute.(schema.MapNestedAttribute).NestedObject.Attributes
			}
			var err error
			entries, err = objectCases(ctx, p.Item.Properties, nested, nil)
			if err != nil {
				return nil, err
			}
			elementType = types.ObjectType{AttrTypes: interpreter.AttributeTypes(nested)}
		}
		result := []example{}
		if p.Type == "list" {
			result = append(result, example{"empty", types.ListValueMust(elementType, []attr.Value{})})
		} else {
			result = append(result, example{"empty", types.MapValueMust(elementType, map[string]attr.Value{})})
		}
		if p.MaxItems != nil && *p.MaxItems == 0 {
			return result, nil
		}
		for _, entry := range entries {
			var value attr.Value
			if p.Type == "list" {
				value = types.ListValueMust(elementType, []attr.Value{entry.value})
			} else {
				value = types.MapValueMust(elementType, map[string]attr.Value{"sample": entry.value})
			}
			result = append(result, example{"entry." + entry.name, value})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported sample type %q", p.Type)
	}
}

func nullValue(ctx context.Context, attribute schema.Attribute) (attr.Value, error) {
	typeInfo := attribute.GetType()
	return typeInfo.ValueFromTerraform(ctx, tftypes.NewValue(typeInfo.TerraformType(ctx), nil))
}
func nonempty(value attr.Value) bool {
	if value == nil || value.IsNull() || value.IsUnknown() {
		return false
	}
	switch value := value.(type) {
	case types.String:
		return value.ValueString() != ""
	case types.List:
		return len(value.Elements()) > 0
	case types.Map:
		return len(value.Elements()) > 0
	default:
		return true
	}
}
func firstNonempty(cases []example) (attr.Value, error) {
	for _, candidate := range cases {
		if nonempty(candidate.value) {
			return candidate.value, nil
		}
	}
	return nil, fmt.Errorf("curated constraints require a nonempty value but no sample can represent it")
}

// Satisfy curated sibling conditions or fail rather than skip a property. A
// focused selector keeps its sample and clears incompatible optional dependents.
func conditions(ctx context.Context, properties map[string]interpreter.Property, attributes map[string]schema.Attribute, variants map[string][]example, values map[string]attr.Value, focus string) error {
	for pass := 0; pass <= len(properties); pass++ {
		changed := false
		set := func(name string, value attr.Value) {
			if !values[name].Equal(value) {
				values[name], changed = value, true
			}
		}
		unset := func(name string) error {
			if properties[name].Required {
				return fmt.Errorf("cannot omit required conflicting property %s", name)
			}
			value, err := nullValue(ctx, attributes[name])
			if err == nil {
				set(name, value)
			}
			return err
		}
		for _, name := range names(properties) {
			p := properties[name]
			if !values[name].IsNull() {
				for _, peer := range p.ConflictsWith {
					if !values[peer].IsNull() {
						remove := peer
						if peer == focus {
							remove = name
						}
						if err := unset(remove); err != nil {
							return err
						}
					}
				}
				if condition := p.RequiresValue; condition != nil && !values[name].IsNull() {
					value := types.StringValue(condition.Value)
					if condition.Field == focus && !values[focus].Equal(value) {
						if err := unset(name); err != nil {
							return err
						}
					} else {
						set(condition.Field, value)
					}
				}
			}
			if condition := p.RequiredWhen; condition != nil && values[condition.Field].Equal(types.StringValue(condition.Value)) && !nonempty(values[name]) {
				value, err := firstNonempty(variants[name])
				if err != nil {
					return err
				}
				set(name, value)
			}
		}
		if !changed {
			return nil
		}
	}
	return fmt.Errorf("curated conditions did not converge")
}
