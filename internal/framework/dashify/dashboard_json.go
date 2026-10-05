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
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type dashboardJSONSemanticEqualityModifier struct{}

type dashifyLayoutLengthSemanticEqualityModifier struct{}

func (dashifyLayoutLengthSemanticEqualityModifier) Description(_ context.Context) string {
	return "Treats finite numeric layout lengths as unchanged when their numeric values match."
}

func (modifier dashifyLayoutLengthSemanticEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return modifier.Description(ctx)
}

func (dashifyLayoutLengthSemanticEqualityModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	stateNumber, stateErr := strconv.ParseFloat(strings.TrimSpace(req.StateValue.ValueString()), 64)
	configNumber, configErr := strconv.ParseFloat(strings.TrimSpace(req.ConfigValue.ValueString()), 64)
	if stateErr == nil && configErr == nil && !math.IsNaN(stateNumber) && !math.IsInf(stateNumber, 0) && stateNumber == configNumber {
		resp.PlanValue = req.StateValue
	}
}

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

func dashifyFiniteNumber(raw any) (float64, bool) {
	var number float64
	switch value := raw.(type) {
	case float64:
		number = value
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0, false
		}
		number = parsed
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

func dashifyExactInt64(raw any) (int64, bool) {
	switch value := raw.(type) {
	case int64:
		return value, true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value ||
			value >= math.Exp2(63) || value < -math.Exp2(63) {
			return 0, false
		}
		return int64(value), true
	case json.Number:
		rational, ok := new(big.Rat).SetString(value.String())
		if !ok || !rational.IsInt() || !rational.Num().IsInt64() {
			return 0, false
		}
		return rational.Num().Int64(), true
	default:
		return 0, false
	}
}
