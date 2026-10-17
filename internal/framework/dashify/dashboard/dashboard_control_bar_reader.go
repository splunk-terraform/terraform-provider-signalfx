// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A reader consumes modeled fields and retains the first error, so callers can
// read a complete control while preserving field-order diagnostics.
type dashifyControlReader struct {
	object map[string]any
	path   string
	err    error
}

func (r *dashifyControlReader) string(key string) types.String {
	raw, ok := r.object[key]
	if !ok || r.err != nil {
		return types.StringNull()
	}
	value, ok := raw.(string)
	if !ok {
		r.err = fmt.Errorf("%s.%s is %v (%T) rather than a string", r.path, key, raw, raw)
		return types.StringNull()
	}
	delete(r.object, key)
	return types.StringValue(value)
}

// The UI saves empty optional labels and descriptions. Treat them as omitted
// so read state matches the non-empty schema and generated import config.
func (r *dashifyControlReader) optionalString(key string) types.String {
	value := r.string(key)
	if value.ValueString() == "" {
		return types.StringNull()
	}
	return value
}

func (r *dashifyControlReader) bool(key string) types.Bool {
	raw, ok := r.object[key]
	if !ok || r.err != nil {
		return types.BoolNull()
	}
	value, ok := raw.(bool)
	if !ok {
		r.err = fmt.Errorf("%s.%s is %v (%T) rather than a boolean", r.path, key, raw, raw)
		return types.BoolNull()
	}
	delete(r.object, key)
	return types.BoolValue(value)
}

func (r *dashifyControlReader) int64(key string) types.Int64 {
	raw, ok := r.object[key]
	if !ok || r.err != nil {
		return types.Int64Null()
	}
	value, valid := dashifyExactInt64(raw)
	if !valid {
		r.err = fmt.Errorf("%s.%s is %v (%T) rather than an integer", r.path, key, raw, raw)
		return types.Int64Null()
	}
	delete(r.object, key)
	return types.Int64Value(value)
}

func (r *dashifyControlReader) stringList(key string) []types.String {
	raw, ok := r.object[key]
	if !ok || r.err != nil {
		return nil
	}
	values, ok := raw.([]any)
	if !ok {
		r.err = fmt.Errorf("%s.%s is %v (%T) rather than a list", r.path, key, raw, raw)
		return nil
	}
	result := make([]types.String, len(values))
	for i, rawValue := range values {
		value, ok := rawValue.(string)
		if !ok {
			r.err = fmt.Errorf("%s.%s.%d is %v (%T) rather than a string", r.path, key, i, rawValue, rawValue)
			return nil
		}
		result[i] = types.StringValue(value)
	}
	delete(r.object, key)
	return result
}

func (r *dashifyControlReader) filterSetEntries(key string) ([]dashifyFilterSetEntryModel, []string) {
	raw, ok := r.object[key]
	if !ok || r.err != nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		r.err = fmt.Errorf("%s.%s is %v (%T) rather than a list", r.path, key, raw, raw)
		return nil, nil
	}
	var result []dashifyFilterSetEntryModel
	var leftovers []string
	for i, rawValue := range values {
		entryPath := fmt.Sprintf("%s.%s.%d", r.path, key, i)
		entry, ok := rawValue.(map[string]any)
		if !ok {
			r.err = fmt.Errorf("%s is %T rather than an object", entryPath, rawValue)
			return nil, nil
		}
		reader := dashifyControlReader{object: entry, path: entryPath}
		filter := dashifyFilterSetEntryModel{}
		filter.Property = reader.string("key")
		if reader.err == nil && filter.Property.ValueString() == "" {
			reader.err = fmt.Errorf("%s.key must be a non-empty string", entryPath)
		}
		filter.Values = reader.stringList("values")
		if reader.err == nil && filter.Values == nil {
			reader.err = fmt.Errorf("%s.values must be a list", entryPath)
		}
		filter.Negated = reader.bool("negated")
		filter.Disabled = reader.bool("disabled")
		filter.MatchMissing = reader.bool("matchMissing")
		if reader.err != nil {
			r.err = reader.err
			return nil, nil
		}
		// The filter entry itself is an independently modeled object, so its
		// unknown fields are reported at the precise nested path.
		result = append(result, filter)
		leftovers = append(leftovers, dashifyLeftovers(entryPath, entry)...)
	}
	delete(r.object, key)
	return result, leftovers
}
