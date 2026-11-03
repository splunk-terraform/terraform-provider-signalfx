// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeObject preserves JSON numbers and requires a non-null object.
func DecodeObject(raw string) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("chart spec must be a JSON object")
	}
	return object, nil
}
