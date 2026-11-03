// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Numeric, duration, and consuming parsers decode Terraform values and
// leave unmodeled fields available for precise import warnings.
package interpreter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

// takeLeaf removes and returns the value at path. Null decodes as an unset
// Terraform value. A missing intermediate object is optional; an existing
// parent of another type is malformed configuration.
func takeLeaf(root map[string]any, path []string) (any, bool, error) {
	m := root
	parents := []map[string]any{root}
	for index, seg := range path[:len(path)-1] {
		value, present := m[seg]
		if !present {
			return nil, false, nil
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, false, typeError(path[:index+1], "an object", value)
		}
		m = next
		parents = append(parents, m)
	}

	leaf := path[len(path)-1]
	value, ok := m[leaf]
	if !ok {
		return nil, false, nil
	}
	delete(m, leaf)
	for index := len(parents) - 1; index > 0; index-- {
		if len(parents[index]) != 0 {
			break
		}
		delete(parents[index-1], path[index-1])
	}

	return value, value != nil, nil
}

// decodeField removes the value at path, returning null when absent, and converts
// present values with the type-specific parser.
func decodeField[T any](root map[string]any, path []string, null T, convert func(raw any) (T, error)) (T, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return null, err
	}
	return convert(raw)
}

func takeString(root map[string]any, path []string) (types.String, error) {
	return decodeField(root, path, types.StringNull(), func(raw any) (types.String, error) {
		value, ok := raw.(string)
		if !ok {
			return types.StringNull(), typeError(path, "a string", raw)
		}
		return types.StringValue(value), nil
	})
}

func takeBool(root map[string]any, path []string) (types.Bool, error) {
	return decodeField(root, path, types.BoolNull(), func(raw any) (types.Bool, error) {
		value, ok := raw.(bool)
		if !ok {
			return types.BoolNull(), typeError(path, "true or false", raw)
		}
		return types.BoolValue(value), nil
	})
}

// takeInt64 decodes a JSON integer, rejecting fractional numbers and overflow.
func takeInt64(root map[string]any, path []string) (types.Int64, error) {
	return decodeField(root, path, types.Int64Null(), func(raw any) (types.Int64, error) {
		value, err := jsonInt64(raw)
		if err != nil {
			return types.Int64Null(), fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		return types.Int64Value(value), nil
	})
}

// takeFloat64 decodes a JSON number into a Terraform Float64 value.
func takeFloat64(root map[string]any, path []string) (types.Float64, error) {
	return decodeField(root, path, types.Float64Null(), func(raw any) (types.Float64, error) {
		value, err := jsonFloat64(raw)
		if err != nil {
			return types.Float64Null(), fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		return types.Float64Value(value), nil
	})
}

func takeStringList(root map[string]any, path []string) ([]attr.Value, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return nil, err
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, typeError(path, "a list of strings", raw)
	}
	out := make([]attr.Value, len(values))
	for i, rawValue := range values {
		value, ok := rawValue.(string)
		if !ok {
			return nil, typeError(append(path, strconv.Itoa(i)), "a string", rawValue)
		}
		out[i] = types.StringValue(value)
	}
	return out, nil
}

func takeObjectList(root map[string]any, path []string, parse func(map[string]any) (types.Object, error)) ([]attr.Value, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return nil, err
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, typeError(path, "a list of objects", raw)
	}
	out := make([]attr.Value, len(values))
	residual := make([]any, len(values))
	hasResidual := false
	for i, rawValue := range values {
		value, ok := rawValue.(map[string]any)
		if !ok {
			return nil, typeError(append(path, strconv.Itoa(i)), "an object", rawValue)
		}
		parsed, err := parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s.%d: %w", strings.Join(path, "."), i, err)
		}
		if len(value) != 0 {
			residual[i] = value
			hasResidual = true
		} else {
			residual[i] = consumedListItem{}
		}
		out[i] = parsed
	}
	if hasResidual {
		setPath(root, path, residual)
	}
	return out, nil
}

// consumedListItem preserves indices when only some parsed list items carry
// unmodeled fields. It cannot be confused with a stored empty JSON object.
type consumedListItem struct{}

func takeObjectMap(root map[string]any, path []string, parse func(map[string]any) (types.Object, error)) (map[string]attr.Value, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return nil, err
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, typeError(path, "an object map", raw)
	}
	out := make(map[string]attr.Value, len(values))
	residual := map[string]any{}
	for key, rawValue := range values {
		value, ok := rawValue.(map[string]any)
		if !ok {
			return nil, typeError(append(path, key), "an object", rawValue)
		}
		parsed, err := parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", strings.Join(path, "."), key, err)
		}
		if len(value) != 0 {
			residual[key] = value
		}
		out[key] = parsed
	}
	if len(residual) != 0 {
		setPath(root, path, residual)
	}
	return out, nil
}

func takeObject(root map[string]any, path []string, null types.Object, parse func(map[string]any) (types.Object, error)) (types.Object, error) {
	return decodeField(root, path, null, func(raw any) (types.Object, error) {
		value, ok := raw.(map[string]any)
		if !ok {
			return null, typeError(path, "an object", raw)
		}
		parsed, err := parse(value)
		if err != nil {
			return null, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		if len(value) != 0 {
			setPath(root, path, value)
		}
		return parsed, nil
	})
}

// takeOneOf decodes a scalar or object variant, leaving unrecognized object
// fields available for unsupported-field warnings.
func takeOneOf(root map[string]any, path []string, null types.Object, parse func(raw any) (types.Object, error)) (types.Object, error) {
	return decodeField(root, path, null, func(raw any) (types.Object, error) {
		parsed, err := parse(raw)
		if err != nil {
			return null, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		if obj, ok := raw.(map[string]any); ok && len(obj) != 0 {
			setPath(root, path, obj)
		}
		return parsed, nil
	})
}

func relativeDurationSpec(value string) (map[string]any, bool) {
	milliseconds, ok := relativeDurationMillis(value)
	if !ok {
		return nil, false
	}
	return map[string]any{"type": "relative", "range": milliseconds, "rangeEnd": int64(0)}, true
}

func relativeDurationMillis(value string) (int64, bool) {
	milliseconds, err := common.FromTimeRangeToMilliseconds(value)
	return int64(milliseconds), err == nil && milliseconds > 0 && milliseconds%1000 == 0
}

func takeRelativeDuration(root map[string]any, path []string) (types.String, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return types.StringNull(), err
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return types.StringNull(), typeError(path, "a relative time object", raw)
	}
	typeValue, ok := value["type"].(string)
	if !ok || typeValue != "relative" {
		return types.StringNull(), fmt.Errorf("%s.type is %v, want relative", strings.Join(path, "."), value["type"])
	}
	rangeValue, err := jsonInt64(value["range"])
	if err != nil {
		return types.StringNull(), fmt.Errorf("%s.range: %w", strings.Join(path, "."), err)
	}
	rangeEnd, err := jsonInt64(value["rangeEnd"])
	if err != nil || rangeEnd != 0 {
		return types.StringNull(), fmt.Errorf("%s.rangeEnd is %v, want 0", strings.Join(path, "."), value["rangeEnd"])
	}
	if rangeValue <= 0 {
		return types.StringNull(), fmt.Errorf("%s.range is %dms, want a positive duration", strings.Join(path, "."), rangeValue)
	}
	delete(value, "type")
	delete(value, "range")
	delete(value, "rangeEnd")
	if len(value) != 0 {
		setPath(root, path, value)
	}
	for _, suffix := range []rune{'w', 'd', 'h', 'm', 's'} {
		if millis, ok := common.TimeRangeUnitMilliseconds(suffix); ok && rangeValue%int64(millis) == 0 {
			return types.StringValue("-" + strconv.FormatInt(rangeValue/int64(millis), 10) + string(suffix)), nil
		}
	}
	return types.StringNull(), fmt.Errorf("%s.range is %dms, which cannot be represented as whole seconds", strings.Join(path, "."), rangeValue)
}

func jsonInt64(raw any) (int64, error) {
	number, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%v (%T) is not a number", raw, raw)
	}
	return number.Int64()
}

func jsonFloat64(raw any) (float64, error) {
	number, ok := raw.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%v (%T) is not a number", raw, raw)
	}
	return number.Float64()
}

func typeError(path []string, want string, got any) error {
	return fmt.Errorf("%s holds %v (%T), want %s", strings.Join(path, "."), got, got, want)
}

// Leftovers lists fields the typed parser did not claim, including empty
// collections and nulls. Decode removes its declared empty root containers
// first. Residual list items retain indices.
func Leftovers(prefix string, node map[string]any) []string {
	var out []string
	collectLeftovers(prefix, node, &out)
	sort.Strings(out)

	return out
}

func collectLeftovers(prefix string, node map[string]any, out *[]string) {
	for key, value := range node {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		switch typed := value.(type) {
		case nil:
			*out = append(*out, path)
		case map[string]any:
			if len(typed) == 0 {
				*out = append(*out, path)
			} else {
				collectLeftovers(path, typed, out)
			}
		case []any:
			if len(typed) == 0 {
				*out = append(*out, path)
				continue
			}
			for index, child := range typed {
				childPath := path + "." + strconv.Itoa(index)
				switch child := child.(type) {
				case consumedListItem:
					continue
				case map[string]any:
					if len(child) == 0 {
						*out = append(*out, childPath)
					} else {
						collectLeftovers(childPath, child, out)
					}
				default:
					*out = append(*out, childPath)
				}
			}
		default:
			*out = append(*out, path)
		}
	}
}
