// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Converts Terraform strings to polymorphic layout values, using JSON for objects and coordinate arrays.
func dashifyLayoutValue(value types.String, coordinate bool) (any, bool, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, false, nil
	}

	text := value.ValueString()
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var decoded any
		decoder := json.NewDecoder(strings.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			return nil, false, fmt.Errorf("must contain valid JSON: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return nil, false, fmt.Errorf("must contain exactly one JSON value")
			}
			return nil, false, fmt.Errorf("must contain valid JSON: %w", err)
		}
		normalized, err := normalizeDashifyAdvancedLayoutValue(decoded, coordinate)
		if err != nil {
			return nil, false, err
		}
		return normalized, true, nil
	}

	if number, err := strconv.ParseFloat(trimmed, 64); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		return number, true, nil
	}
	return text, true, nil
}

func normalizeDashifyAdvancedLayoutValue(value any, coordinate bool) (any, error) {
	switch value := value.(type) {
	case map[string]any:
		return normalizeDashifyClampedLength(value)
	case []any:
		if !coordinate {
			return nil, fmt.Errorf("JSON arrays are supported only for x and y coordinates")
		}
		normalized := make([]any, len(value))
		for i, part := range value {
			length, err := normalizeDashifyLength(part)
			if err != nil {
				return nil, fmt.Errorf("coordinate part %d: %w", i, err)
			}
			normalized[i] = length
		}
		return normalized, nil
	default:
		if coordinate {
			return nil, fmt.Errorf("JSON coordinate values must be a clamped object or array")
		}
		return nil, fmt.Errorf("JSON layout values must be a clamped object")
	}
}

func normalizeDashifyLength(value any) (any, error) {
	switch value := value.(type) {
	case string:
		return value, nil
	case json.Number:
		return normalizeDashifyNumber(value)
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("length numbers must be finite")
		}
		return value, nil
	case map[string]any:
		return normalizeDashifyClampedLength(value)
	default:
		return nil, fmt.Errorf("length must be a number, string, or clamped object, got %T", value)
	}
}

func normalizeDashifyClampedLength(value map[string]any) (map[string]any, error) {
	if _, ok := value["value"]; !ok {
		return nil, fmt.Errorf("clamped layout objects must contain value")
	}
	for key := range value {
		if key != "value" && key != "min" && key != "max" {
			return nil, fmt.Errorf("clamped layout objects do not support %q", key)
		}
	}

	normalized := make(map[string]any, len(value))
	for _, key := range []string{"value", "min", "max"} {
		raw, ok := value[key]
		if !ok {
			continue
		}
		length, err := normalizeDashifySimpleLength(raw)
		if err != nil {
			return nil, fmt.Errorf("clamped %s: %w", key, err)
		}
		normalized[key] = length
	}
	return normalized, nil
}

func normalizeDashifySimpleLength(value any) (any, error) {
	switch value := value.(type) {
	case string:
		return value, nil
	case json.Number:
		return normalizeDashifyNumber(value)
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("length numbers must be finite")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("must be a number or string, got %T", value)
	}
}

func normalizeDashifyNumber(value json.Number) (float64, error) {
	number, err := value.Float64()
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("length number %q is not finite", value)
	}
	return number, nil
}

func dashifyLayoutString(item map[string]any, key string, coordinate bool) (types.String, error) {
	value, ok := item[key]
	if !ok {
		return types.StringNull(), nil
	}
	delete(item, key)

	switch value := value.(type) {
	case string:
		return types.StringValue(value), nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return types.StringNull(), fmt.Errorf("has %s %v rather than a finite number", key, value)
		}
		return types.StringValue(strconv.FormatFloat(value, 'f', -1, 64)), nil
	case json.Number:
		number, err := normalizeDashifyNumber(value)
		if err != nil {
			return types.StringNull(), fmt.Errorf("has invalid %s: %w", key, err)
		}
		return types.StringValue(strconv.FormatFloat(number, 'f', -1, 64)), nil
	case map[string]any, []any:
		normalized, err := normalizeDashifyAdvancedLayoutValue(value, coordinate)
		if err != nil {
			return types.StringNull(), fmt.Errorf("has invalid %s: %w", key, err)
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return types.StringNull(), fmt.Errorf("encode %s: %w", key, err)
		}
		return types.StringValue(string(encoded)), nil
	default:
		return types.StringNull(), fmt.Errorf("has %s %v (%T) rather than a layout length", key, value, value)
	}
}
