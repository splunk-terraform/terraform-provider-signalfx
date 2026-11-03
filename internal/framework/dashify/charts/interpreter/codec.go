// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (e *Engine) Encode(ctx context.Context, name string, value types.Object) (map[string]any, diag.Diagnostics) {
	diagnostics := e.Validate(name, value)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	spec, err := encodeObject(e.definitions[name].Properties, value)
	if err != nil {
		diagnostics.AddError("Invalid chart value", fmt.Sprintf("%s: %v", name, err))
		return nil, diagnostics
	}
	spec["<"+e.definitions[name].Element+">"] = []any{}
	for _, group := range e.roots[name] {
		if _, exists := spec[group]; !exists {
			spec[group] = map[string]any{}
		}
	}
	return spec, diagnostics
}

func encodeObject(properties map[string]property, value types.Object) (map[string]any, error) {
	result := map[string]any{}
	for _, name := range keys(properties) {
		v := value.Attributes()[name]
		if !present(v) {
			continue
		}
		p := properties[name]
		wire, err := encodeValue(p, v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if p.Mapping.Transform == transformNegate {
			wire = !wire.(bool)
		}
		if p.Mapping.Transform == transformRelativeDuration {
			wire, _ = relativeDurationSpec(wire.(string))
		}
		setPath(result, p.Mapping.Path, wire)
	}
	return result, nil
}

func encodeValue(p property, value attr.Value) (any, error) {
	switch v := value.(type) {
	case types.String:
		return v.ValueString(), nil
	case types.Bool:
		return v.ValueBool(), nil
	case types.Int64:
		return v.ValueInt64(), nil
	case types.Float64:
		return v.ValueFloat64(), nil
	case types.Object:
		if p.Type == propertyTypeOneOf {
			for _, name := range keys(p.Variants) {
				variant := p.Variants[name]
				if variant.Type != propertyTypeBlock {
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
			return nil, nil
		}
		children, _ := childProperties(p)
		return encodeObject(children, v)
	case types.List:
		result := make([]any, len(v.Elements()))
		for index, child := range v.Elements() {
			wire, err := encodeValue(property{Type: propertyTypeBlock, Properties: p.Item.Properties}, child)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", index, err)
			}
			result[index] = wire
		}
		return result, nil
	case types.Map:
		result := map[string]any{}
		for key, child := range v.Elements() {
			wire, err := encodeValue(property{Type: propertyTypeBlock, Properties: p.Item.Properties}, child)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", key, err)
			}
			result[key] = wire
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported chart value %T", value)
	}
}

// Decode consumes spec to build Terraform values and report unmodeled fields.
func (e *Engine) Decode(ctx context.Context, name string, spec map[string]any, prior types.Object) (types.Object, []string, diag.Diagnostics) {
	attributes, exists := e.attributes[name]
	var diagnostics diag.Diagnostics
	if !exists {
		diagnostics.AddError("Invalid chart", fmt.Sprintf("Unknown chart %q.", name))
		return types.ObjectNull(nil), nil, diagnostics
	}
	null := types.ObjectNull(attributeTypes(attributes))
	var priorValues map[string]attr.Value
	if present(prior) {
		// Only top-level relative durations preserve their configured spelling.
		priorValues = prior.Attributes()
	}
	for _, group := range e.roots[name] {
		if _, ok := spec[group].(map[string]any); !ok {
			diagnostics.AddError("Invalid chart document", group+" must be an object.")
			return null, nil, diagnostics
		}
	}
	tag := "<" + e.definitions[name].Element + ">"
	if args, valid := spec[tag].([]any); !valid || len(args) != 0 {
		diagnostics.AddError("Invalid chart document", "Element "+tag+" must have an empty argument list.")
		return null, nil, diagnostics
	}
	delete(spec, tag)
	value, err := decodeObject(ctx, e.definitions[name].Properties, attributeTypes(attributes), spec, priorValues)
	if err != nil {
		diagnostics.AddError("Invalid chart document", err.Error())
		return null, nil, diagnostics
	}
	// Decode does not apply configuration rules, so representable server drift
	// never blocks refresh or deletion. Callers decide whether to report it.
	for _, group := range e.roots[name] {
		if remaining, ok := spec[group].(map[string]any); ok && len(remaining) == 0 {
			delete(spec, group)
		}
	}
	return value, Leftovers("", spec), diagnostics
}

func decodeObject(ctx context.Context, properties map[string]property, attributeTypes map[string]attr.Type, spec map[string]any, priorValues map[string]attr.Value) (types.Object, error) {
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

func decodeValue(ctx context.Context, p property, valueType attr.Type, spec map[string]any, prior attr.Value) (attr.Value, error) {
	//exhaustive:enforce
	switch p.Type {
	case propertyTypeString, propertyTypeEnum:
		if p.Mapping.Transform == transformRelativeDuration {
			value, err := takeRelativeDuration(spec, p.Mapping.Path)
			if old, ok := prior.(types.String); ok && present(old) && present(value) {
				oldMillis, valid := relativeDurationMillis(old.ValueString())
				newMillis, _ := relativeDurationMillis(value.ValueString())
				if valid && oldMillis == newMillis {
					value = old
				}
			}
			return value, err
		}
		return takeString(spec, p.Mapping.Path)
	case propertyTypeBool:
		value, err := takeBool(spec, p.Mapping.Path)
		if p.Mapping.Transform == transformNegate && present(value) {
			value = types.BoolValue(!value.ValueBool())
		}
		return value, err
	case propertyTypeNumber:
		return takeInt64(spec, p.Mapping.Path)
	case propertyTypeFloat:
		return takeFloat64(spec, p.Mapping.Path)
	case propertyTypeBlock, propertyTypeOneOf:
		objectType := valueType.(types.ObjectType)
		decoder := objectDecoder{ctx: ctx, definition: p, objectType: objectType}
		null := types.ObjectNull(objectType.AttrTypes)
		if p.Type == propertyTypeOneOf {
			return takeOneOf(spec, p.Mapping.Path, null, decoder.decodeOneOf)
		}
		return takeObject(spec, p.Mapping.Path, null, decoder.decodeObject)
	case propertyTypeList, propertyTypeMap:
		if p.Type == propertyTypeList && p.Item.Type == propertyTypeString {
			values, err := takeStringList(spec, p.Mapping.Path)
			if err != nil || values == nil {
				return types.ListNull(types.StringType), err
			}
			value, d := types.ListValue(types.StringType, values)
			if d.HasError() {
				return value, fmt.Errorf("list conversion: %v", d)
			}
			return value, nil
		}
		if p.Type == propertyTypeList {
			objectType := valueType.(types.ListType).ElemType.(types.ObjectType)
			decoder := objectDecoder{ctx: ctx, definition: p, objectType: objectType}
			values, err := takeObjectList(spec, p.Mapping.Path, decoder.decodeObject)
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
		decoder := objectDecoder{ctx: ctx, definition: p, objectType: objectType}
		values, err := takeObjectMap(spec, p.Mapping.Path, decoder.decodeObject)
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

type objectDecoder struct {
	ctx        context.Context
	definition property
	objectType types.ObjectType
}

func (d objectDecoder) decodeObject(raw map[string]any) (types.Object, error) {
	children, _ := childProperties(d.definition)
	return decodeObject(d.ctx, children, d.objectType.AttrTypes, raw, nil)
}

func (d objectDecoder) decodeOneOf(raw any) (types.Object, error) {
	return decodeOneOf(d.ctx, d.definition, raw, d.objectType)
}

func decodeOneOf(ctx context.Context, p property, raw any, objectType types.ObjectType) (types.Object, error) {
	children, _ := childProperties(p)
	if object, ok := raw.(map[string]any); ok {
		blockFields := map[string]property{}
		for _, variant := range p.Variants {
			if variant.Type == propertyTypeBlock {
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
		if variant.Type == propertyTypeBlock {
			continue
		}
		if _, ok := raw.(string); ok && (variant.Type == propertyTypeString || variant.Type == propertyTypeEnum) {
			object := map[string]any{name: raw}
			return decodeObject(ctx, map[string]property{name: children[name]}, objectType.AttrTypes, object, nil)
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
