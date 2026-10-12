// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Simple sizes are finite numbers or strings such as "1/2" and "100%".
func dashifyLayoutValue(value types.String) (any, bool, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, false, nil
	}
	text := value.ValueString()
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return nil, false, fmt.Errorf("JSON objects and arrays are not supported in typed layout sizes; use signalfx_observability_template for advanced layout")
	}
	if number, err := strconv.ParseFloat(trimmed, 64); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		return number, true, nil
	}
	return text, true, nil
}

// Leave advanced or unknown values in the source object. The caller reports
// them as unmodeled and blocks a full-document update that would erase them.
func dashifyLayoutString(item map[string]any, key string) (types.String, error) {
	value, ok := item[key]
	if !ok {
		return types.StringNull(), nil
	}
	switch value := value.(type) {
	case string:
		delete(item, key)
		return types.StringValue(value), nil
	case json.Number:
		number, ok := dashifyFiniteNumber(value)
		if !ok {
			return types.StringNull(), fmt.Errorf("has invalid %s: length number %q is not finite", key, value)
		}
		delete(item, key)
		return types.StringValue(strconv.FormatFloat(number, 'f', -1, 64)), nil
	default:
		return types.StringNull(), nil
	}
}
