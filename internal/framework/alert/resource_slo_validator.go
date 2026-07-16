// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/signalfx/signalfx-go/slo"
)

func (model resourceSLOModel) validate(ctx context.Context) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	inputs := detectorListElements[sloInputModel](ctx, model.Input, &diagnostics)
	if model.Input.IsNull() || (!model.Input.IsUnknown() && len(inputs) != 1) {
		diagnostics.AddAttributeError(path.Root("input"), "Invalid SLO input", "Exactly one input block is required.")
	}
	targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
	if model.Target.IsNull() || (!model.Target.IsUnknown() && len(targets) != 1) {
		diagnostics.AddAttributeError(path.Root("target"), "Invalid SLO target", "Exactly one target block is required.")
		return diagnostics
	}
	if model.Target.IsUnknown() || len(targets) != 1 || targets[0].Type.IsUnknown() {
		return diagnostics
	}

	target := targets[0]
	switch target.Type.ValueString() {
	case slo.RollingWindowTarget:
		if target.CompliancePeriod.IsNull() {
			diagnostics.AddAttributeError(path.Root("target"), "Missing rolling compliance period", "RollingWindow targets require compliance_period.")
		}
		if !target.CycleType.IsNull() || (!target.CycleStart.IsNull() && !target.CycleStart.IsUnknown()) {
			diagnostics.AddAttributeError(path.Root("target"), "Invalid rolling target fields", "RollingWindow targets cannot configure cycle_type or cycle_start.")
		}
	case slo.CalendarWindowTarget:
		if target.CycleType.IsNull() {
			diagnostics.AddAttributeError(path.Root("target"), "Missing calendar cycle type", "CalendarWindow targets require cycle_type.")
		}
		if !target.CompliancePeriod.IsNull() {
			diagnostics.AddAttributeError(path.Root("target"), "Invalid calendar target fields", "CalendarWindow targets cannot configure compliance_period.")
		}
	}

	if target.AlertRules.IsUnknown() {
		return diagnostics
	}
	alertRules := detectorListElements[sloAlertRuleModel](ctx, target.AlertRules, &diagnostics)
	if target.AlertRules.IsNull() || (!target.AlertRules.IsUnknown() && len(alertRules) == 0) {
		diagnostics.AddAttributeError(path.Root("target"), "Missing SLO alert rules", "At least one alert_rule block is required.")
		return diagnostics
	}
	seen := make(map[string]struct{}, len(alertRules))
	hasBreach := false
	hasUnknownType := false
	for index, alertRule := range alertRules {
		if alertRule.Type.IsUnknown() {
			hasUnknownType = true
			continue
		}
		ruleType := alertRule.Type.ValueString()
		if _, exists := seen[ruleType]; exists {
			diagnostics.AddAttributeError(path.Root("target"), "Duplicate SLO alert rule", fmt.Sprintf("Alert rule type %q can be configured only once.", ruleType))
		}
		seen[ruleType] = struct{}{}
		hasBreach = hasBreach || ruleType == slo.BreachRule
		rules := detectorListElements[sloRuleModel](ctx, alertRule.Rules, &diagnostics)
		if alertRule.Rules.IsNull() || (!alertRule.Rules.IsUnknown() && len(rules) == 0) {
			diagnostics.AddAttributeError(path.Root("target"), "Missing detector rule", fmt.Sprintf("alert_rule block %d must contain at least one rule block.", index))
		}
	}
	if !hasBreach && !hasUnknownType {
		diagnostics.AddAttributeError(path.Root("target"), "Missing BREACH alert rule", "Every SLO target requires one BREACH alert_rule block.")
	}
	return diagnostics
}

func (model resourceSLOModel) hasUnknownRequiredValues(ctx context.Context) bool {
	if model.Name.IsUnknown() || model.Type.IsUnknown() || model.Input.IsUnknown() || model.Target.IsUnknown() {
		return true
	}
	var diagnostics diag.Diagnostics
	inputs := detectorListElements[sloInputModel](ctx, model.Input, &diagnostics)
	targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
	if diagnostics.HasError() || len(inputs) != 1 || len(targets) != 1 ||
		inputs[0].ProgramText.IsUnknown() || targets[0].Type.IsUnknown() || targets[0].SLO.IsUnknown() ||
		targets[0].AlertRules.IsUnknown() {
		return true
	}
	alertRules := detectorListElements[sloAlertRuleModel](ctx, targets[0].AlertRules, &diagnostics)
	for _, alertRule := range alertRules {
		if alertRule.Type.IsUnknown() || alertRule.Rules.IsUnknown() {
			return true
		}
		rules := detectorListElements[sloRuleModel](ctx, alertRule.Rules, &diagnostics)
		for _, rule := range rules {
			if rule.Severity.IsUnknown() {
				return true
			}
		}
	}
	return diagnostics.HasError()
}
