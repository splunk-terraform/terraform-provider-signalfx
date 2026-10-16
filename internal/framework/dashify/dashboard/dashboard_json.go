// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"bytes"
	"encoding/json"
	"math"
)

func decodeDashifyJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(target)
}

// Decoded Dashify JSON always uses json.Number, and JSON cannot encode NaN, so
// a successful parse is finite (out-of-range values fail with ErrRange).
func dashifyFiniteNumber(raw any) (float64, bool) {
	number, ok := raw.(json.Number)
	if !ok {
		return 0, false
	}
	value, err := number.Float64()
	return value, err == nil
}

// Accepts integral forms such as 60.0 and 6e1, not just 60.
func dashifyExactInt64(raw any) (int64, bool) {
	value, ok := dashifyFiniteNumber(raw)
	if !ok || value != math.Trunc(value) || value >= math.MaxInt64 || value < math.MinInt64 {
		return 0, false
	}
	return int64(value), true
}
