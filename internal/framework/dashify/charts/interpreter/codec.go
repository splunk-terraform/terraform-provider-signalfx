// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (e *Engine) Encode(ctx context.Context, name string, value types.Object) (map[string]any, diag.Diagnostics) {
	diagnostics := e.Validate(ctx, name, value, true)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	spec := encodeObject(e.definitions[name].Properties, value)
	spec["<"+e.definitions[name].Element+">"] = []any{}
	for _, group := range e.roots[name] {
		if _, exists := spec[group]; !exists {
			spec[group] = map[string]any{}
		}
	}
	return spec, diagnostics
}

func encodeObject(properties map[string]Property, value types.Object) map[string]any {
	result := map[string]any{}
	for _, name := range keys(properties) {
		v := value.Attributes()[name]
		if !present(v) {
			continue
		}
		p := properties[name]
		wire := encodeValue(p, v)
		if p.Mapping.Transform == "negate" {
			wire = !wire.(bool)
		}
		if p.Mapping.Transform == "relative_duration" {
			wire, _ = RelativeDurationSpec(wire.(string))
		}
		setPath(result, p.Mapping.Path, wire)
	}
	return result
}

func encodeValue(p Property, value attr.Value) any {
	switch v := value.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Int64:
		return v.ValueInt64()
	case types.Float64:
		return v.ValueFloat64()
	case types.Object:
		if p.Type == "oneof" {
			for _, name := range keys(p.Variants) {
				variant := p.Variants[name]
				if variant.Type != "block" {
					if present(v.Attributes()[name]) {
						return encodeValue(variant, v.Attributes()[name])
					}
				} else {
					for _, field := range keys(variant.Properties) {
						if present(v.Attributes()[field]) {
							return encodeObject(variant.Properties, v)
						}
					}
				}
			}
			return nil
		}
		children, _ := childProperties(p)
		return encodeObject(children, v)
	case types.List:
		result := make([]any, len(v.Elements()))
		for index, child := range v.Elements() {
			result[index] = encodeValue(Property{Type: "block", Properties: p.Item.Properties}, child)
		}
		return result
	case types.Map:
		result := map[string]any{}
		for key, child := range v.Elements() {
			result[key] = encodeValue(Property{Type: "block", Properties: p.Item.Properties}, child)
		}
		return result
	default:
		panic(fmt.Sprintf("unsupported chart value %T", value))
	}
}

func (e *Engine) Decode(ctx context.Context, name string, spec map[string]any, prior types.Object) (types.Object, []string, diag.Diagnostics) {
	attributes, exists := e.attributes[name]
	var diagnostics diag.Diagnostics
	if !exists {
		diagnostics.AddError("Invalid chart", fmt.Sprintf("Unknown chart %q.", name))
		return types.ObjectNull(nil), nil, diagnostics
	}
	null := types.ObjectNull(AttributeTypes(attributes))
	var priorValues map[string]attr.Value
	if prior.Type(ctx).Equal(null.Type(ctx)) && present(prior) {
		// Only top-level relative durations preserve their configured spelling.
		priorValues = prior.Attributes()
	}
	work := clone(spec).(map[string]any)
	for _, group := range e.roots[name] {
		if _, ok := work[group].(map[string]any); !ok {
			diagnostics.AddError("Invalid chart document", group+" must be an object.")
			return null, nil, diagnostics
		}
	}
	tag := "<" + e.definitions[name].Element + ">"
	if args, valid := work[tag].([]any); !valid || len(args) != 0 {
		diagnostics.AddError("Invalid chart document", "Element "+tag+" must have an empty argument list.")
		return null, nil, diagnostics
	}
	delete(work, tag)
	value, err := decodeObject(ctx, e.definitions[name].Properties, AttributeTypes(attributes), work, priorValues)
	if err != nil {
		diagnostics.AddError("Invalid chart document", err.Error())
		return null, nil, diagnostics
	}
	// Decode does not apply configuration rules, so representable server drift
	// never blocks refresh or deletion. Callers decide whether to report it.
	for _, group := range e.roots[name] {
		if remaining, ok := work[group].(map[string]any); ok && len(remaining) == 0 {
			delete(work, group)
		}
	}
	return value, Leftovers("", work), diagnostics
}

func decodeObject(ctx context.Context, properties map[string]Property, attributeTypes map[string]attr.Type, spec map[string]any, priorValues map[string]attr.Value) (types.Object, error) {
	values := map[string]attr.Value{}
	for _, name := range keys(attributeTypes) {
		p, exists := properties[name]
		if !exists {
			// Flattened unions leave inactive variants null in Terraform while
			// decoding only the fields belonging to the incoming wire shape.
			t := attributeTypes[name]
			value, err := t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
			if err != nil {
				return types.ObjectNull(attributeTypes), fmt.Errorf("null conversion: %w", err)
			}
			values[name] = value
			continue
		}
		value, err := decodeValue(ctx, p, attributeTypes[name], spec, priorValues[name])
		if err != nil {
			return types.ObjectNull(attributeTypes), err
		}
		values[name] = value
	}
	value, diagnostics := types.ObjectValue(attributeTypes, values)
	if diagnostics.HasError() {
		return value, fmt.Errorf("object conversion: %v", diagnostics)
	}
	return value, nil
}

func decodeValue(ctx context.Context, p Property, valueType attr.Type, spec map[string]any, prior attr.Value) (attr.Value, error) {
	switch p.Type {
	case "string", "enum":
		if p.Mapping.Transform == "relative_duration" {
			value, err := TakeRelativeDuration(spec, p.Mapping.Path)
			if old, ok := prior.(types.String); ok && present(old) && present(value) {
				oldSpec, valid := RelativeDurationSpec(old.ValueString())
				newSpec, _ := RelativeDurationSpec(value.ValueString())
				if valid && reflect.DeepEqual(oldSpec, newSpec) {
					value = old
				}
			}
			return value, err
		}
		return TakeString(spec, p.Mapping.Path)
	case "bool":
		value, err := TakeBool(spec, p.Mapping.Path)
		if p.Mapping.Transform == "negate" && present(value) {
			value = types.BoolValue(!value.ValueBool())
		}
		return value, err
	case "number":
		return TakeInt64(spec, p.Mapping.Path)
	case "float":
		return TakeFloat64(spec, p.Mapping.Path)
	case "block", "oneof":
		objectType := valueType.(types.ObjectType)
		parse := func(raw map[string]any) (types.Object, error) {
			children, _ := childProperties(p)
			return decodeObject(ctx, children, objectType.AttrTypes, raw, nil)
		}
		var object *types.Object
		var err error
		if p.Type == "oneof" {
			object, err = TakeOneOf(spec, p.Mapping.Path, func(raw any) (types.Object, error) { return decodeOneOf(ctx, p, raw, objectType) })
		} else {
			object, err = TakeObject(spec, p.Mapping.Path, parse)
		}
		if object == nil {
			return types.ObjectNull(objectType.AttrTypes), err
		}
		return *object, err
	case "list", "map":
		if p.Type == "list" && p.Item.Type == "string" {
			values, err := TakeStringList(spec, p.Mapping.Path)
			if err != nil || values == nil {
				return types.ListNull(types.StringType), err
			}
			value, d := types.ListValue(types.StringType, values)
			if d.HasError() {
				return value, fmt.Errorf("list conversion: %v", d)
			}
			return value, nil
		}
		children := p.Item.Properties
		if p.Type == "list" {
			objectType := valueType.(types.ListType).ElemType.(types.ObjectType)
			values, err := TakeObjectList(spec, p.Mapping.Path, func(raw map[string]any) (attr.Value, error) {
				return decodeObject(ctx, children, objectType.AttrTypes, raw, nil)
			})
			if err != nil || values == nil {
				return types.ListNull(objectType), err
			}
			value, d := types.ListValue(objectType, values)
			if d.HasError() {
				return value, fmt.Errorf("list conversion: %v", d)
			}
			return value, nil
		}
		objectType := valueType.(types.MapType).ElemType.(types.ObjectType)
		values, err := TakeObjectMap(spec, p.Mapping.Path, func(raw map[string]any) (attr.Value, error) {
			return decodeObject(ctx, children, objectType.AttrTypes, raw, nil)
		})
		if err != nil || values == nil {
			return types.MapNull(objectType), err
		}
		value, d := types.MapValue(objectType, values)
		if d.HasError() {
			return value, fmt.Errorf("map conversion: %v", d)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported chart type %q", p.Type)
	}
}

func decodeOneOf(ctx context.Context, p Property, raw any, objectType types.ObjectType) (types.Object, error) {
	children, _ := childProperties(p)
	if object, ok := raw.(map[string]any); ok {
		blockFields := map[string]Property{}
		for _, variant := range p.Variants {
			if variant.Type == "block" {
				for name, child := range variant.Properties {
					blockFields[name] = child
				}
			}
		}
		if len(blockFields) > 0 {
			return decodeObject(ctx, blockFields, objectType.AttrTypes, object, nil)
		}
	}
	for name, variant := range p.Variants {
		if variant.Type == "block" {
			continue
		}
		if _, ok := raw.(string); ok && (variant.Type == "string" || variant.Type == "enum") {
			object := map[string]any{name: raw}
			return decodeObject(ctx, map[string]Property{name: children[name]}, objectType.AttrTypes, object, nil)
		}
	}
	return types.ObjectNull(objectType.AttrTypes), fmt.Errorf("%T is not a supported oneof shape", raw)
}

func setPath(root map[string]any, path []string, value any) {
	for _, segment := range path[:len(path)-1] {
		child, ok := root[segment].(map[string]any)
		if !ok {
			child = map[string]any{}
			root[segment] = child
		}
		root = child
	}
	root[path[len(path)-1]] = value
}
func clone(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, x := range v {
			out[k] = clone(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = clone(x)
		}
		return out
	default:
		return value
	}
}
