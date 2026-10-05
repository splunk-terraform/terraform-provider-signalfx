// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type dashboardJSONSemanticEqualityModifier struct{}

func (dashboardJSONSemanticEqualityModifier) Description(_ context.Context) string {
	return "Treats JSON content as unchanged when it is semantically equivalent to the prior value."
}

func (modifier dashboardJSONSemanticEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return modifier.Description(ctx)
}

func (dashboardJSONSemanticEqualityModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if dashboardJSONEqual(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

func dashboardJSONEqual(first, second string) bool {
	var firstValue, secondValue any
	if decodeDashifyJSON([]byte(first), &firstValue) != nil || decodeDashifyJSON([]byte(second), &secondValue) != nil {
		return false
	}
	firstJSON, firstErr := json.Marshal(firstValue)
	secondJSON, secondErr := json.Marshal(secondValue)
	return firstErr == nil && secondErr == nil && bytes.Equal(firstJSON, secondJSON)
}

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
