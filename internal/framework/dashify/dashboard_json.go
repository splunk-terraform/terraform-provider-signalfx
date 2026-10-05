// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
)

func decodeDashifyJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("contains multiple JSON values")
		}
		return err
	}
	return nil
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
