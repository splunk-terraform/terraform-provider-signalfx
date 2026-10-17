// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Numeric, duration, and consuming parsers preserve exact Terraform values and
// leave unmodeled fields available for precise import warnings.
package interpreter

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Wire JSON uses json.Number; Encode produces int64 and float64 values.
func exactNumber(raw any) (*big.Rat, bool) {
	var text string
	switch value := raw.(type) {
	case json.Number:
		text = value.String()
		if !json.Valid([]byte(text)) {
			return nil, false
		}
	case int64:
		text = strconv.FormatInt(value, 10)
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, false
		}
		text = strconv.FormatFloat(value, 'g', -1, 64)
	default:
		return nil, false
	}
	return new(big.Rat).SetString(text)
}

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
		if index == 1 && (path[0] == "chart" || path[0] == "datasource" || path[0] == "widget") {
			break
		}
		delete(parents[index-1], path[index-1])
	}

	return value, value != nil, nil
}

// take is TakeString/TakeBool/TakeInt64's shared engine: remove the value at
// path, then hand the raw wire value to convert for the type-specific part.
// null is the value each wrapper returns for "absent" or "wrong type".
// A generic function cannot call types.StringNull() without knowing T.
func take[T any](root map[string]any, path []string, null T, convert func(raw any) (T, error)) (T, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return null, err
	}
	return convert(raw)
}

func TakeString(root map[string]any, path []string) (types.String, error) {
	return take(root, path, types.StringNull(), func(raw any) (types.String, error) {
		value, ok := raw.(string)
		if !ok {
			return types.StringNull(), typeError(path, "a string", raw)
		}
		return types.StringValue(value), nil
	})
}

func TakeBool(root map[string]any, path []string) (types.Bool, error) {
	return take(root, path, types.BoolNull(), func(raw any) (types.Bool, error) {
		value, ok := raw.(bool)
		if !ok {
			return types.BoolNull(), typeError(path, "true or false", raw)
		}
		return types.BoolValue(value), nil
	})
}

// TakeInt64 accepts wire JSON and values produced by Encode, rejecting
// fractional numbers and overflow rather than changing their values.
func TakeInt64(root map[string]any, path []string) (types.Int64, error) {
	return take(root, path, types.Int64Null(), func(raw any) (types.Int64, error) {
		value, err := jsonInt64(raw)
		if err != nil {
			return types.Int64Null(), fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		return types.Int64Value(value), nil
	})
}

// TakeFloat64 accepts both wire and in-process numbers, the same as
// TakeInt64, but keeps any fractional part instead of rejecting it -
// schema's `float` type maps to Float64 here, for fields (a
// threshold bound, a palette range) that are genuinely fractional in real
// documents.
func TakeFloat64(root map[string]any, path []string) (types.Float64, error) {
	return take(root, path, types.Float64Null(), func(raw any) (types.Float64, error) {
		value, err := jsonFloat64(raw)
		if err != nil {
			return types.Float64Null(), fmt.Errorf("%s: %w", strings.Join(path, "."), err)
		}
		return types.Float64Value(value), nil
	})
}

func TakeStringList(root map[string]any, path []string) ([]attr.Value, error) {
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

func TakeObjectList(root map[string]any, path []string, parse func(map[string]any) (attr.Value, error)) ([]attr.Value, error) {
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

func TakeObjectMap(root map[string]any, path []string, parse func(map[string]any) (attr.Value, error)) (map[string]attr.Value, error) {
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

func TakeObject(root map[string]any, path []string, parse func(map[string]any) (types.Object, error)) (*types.Object, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return nil, err
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return nil, typeError(path, "an object", raw)
	}
	parsed, err := parse(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
	}
	if len(value) != 0 {
		setPath(root, path, value)
	}
	return &parsed, nil
}

// TakeOneOf removes the value at path and hands its raw wire shape - a bare
// scalar or an object, whichever the active variant serialized - to parse,
// which recovers the variant itself. Unlike TakeObject, a missing path is simply
// absent rather than a type error, since either shape is valid; when the
// raw value is an object, parse is expected to remove every key it
// recognizes so the Leftovers check below reports only what no variant
// claimed.
func TakeOneOf(root map[string]any, path []string, parse func(raw any) (types.Object, error)) (*types.Object, error) {
	raw, ok, err := takeLeaf(root, path)
	if err != nil || !ok {
		return nil, err
	}

	parsed, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
	}

	if obj, ok := raw.(map[string]any); ok && len(obj) != 0 {
		setPath(root, path, obj)
	}

	return &parsed, nil
}

func RelativeDurationSpec(value string) (map[string]any, bool) {
	milliseconds, ok := relativeDurationMillis(value)
	if !ok {
		return nil, false
	}
	return map[string]any{"type": "relative", "range": milliseconds, "rangeEnd": int64(0)}, true
}

func relativeDurationMillis(value string) (int64, bool) {
	if len(value) < 2 || value[0] < '1' || value[0] > '9' {
		return 0, false
	}
	for _, digit := range value[1 : len(value)-1] {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	number, err := strconv.ParseInt(value[:len(value)-1], 10, 64)
	if err != nil || number <= 0 {
		return 0, false
	}

	var multiplier int64
	switch value[len(value)-1:] {
	case "s":
		multiplier = 1000
	case "m":
		multiplier = 60 * 1000
	case "h":
		multiplier = 60 * 60 * 1000
	case "d":
		multiplier = 24 * 60 * 60 * 1000
	case "w":
		multiplier = 7 * 24 * 60 * 60 * 1000
	default:
		return 0, false
	}
	if number > math.MaxInt64/multiplier {
		return 0, false
	}
	return number * multiplier, true
}

func TakeRelativeDuration(root map[string]any, path []string) (types.String, error) {
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
	units := []struct {
		suffix string
		millis int64
	}{{"w", 7 * 24 * 60 * 60 * 1000}, {"d", 24 * 60 * 60 * 1000}, {"h", 60 * 60 * 1000}, {"m", 60 * 1000}, {"s", 1000}}
	for _, unit := range units {
		if rangeValue%unit.millis == 0 {
			return types.StringValue(strconv.FormatInt(rangeValue/unit.millis, 10) + unit.suffix), nil
		}
	}
	return types.StringNull(), fmt.Errorf("%s.range is %dms, which cannot be represented as whole seconds", strings.Join(path, "."), rangeValue)
}

func jsonInt64(raw any) (int64, error) {
	number, valid := exactNumber(raw)
	if !valid {
		return 0, fmt.Errorf("%v (%T) is not a number", raw, raw)
	}
	if !number.IsInt() {
		return 0, fmt.Errorf("%v is not a whole number", raw)
	}
	integer := number.Num()
	if !integer.IsInt64() {
		return 0, fmt.Errorf("%v is outside the int64 range", raw)
	}
	return integer.Int64(), nil
}

func jsonFloat64(raw any) (float64, error) {
	if value, ok := raw.(float64); ok {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, fmt.Errorf("%v is not a finite number", raw)
		}
		return value, nil
	}
	number, valid := exactNumber(raw)
	if !valid {
		return 0, fmt.Errorf("%v (%T) is not a number", raw, raw)
	}
	value, err := strconv.ParseFloat(number.FloatString(400), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%v cannot be represented as a finite float64", raw)
	}
	roundTrip, roundTripValid := exactNumber(value)
	if !roundTripValid || roundTrip.Cmp(number) != 0 {
		return 0, fmt.Errorf("%v cannot round-trip through float64 without changing value", raw)
	}
	return value, nil
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
