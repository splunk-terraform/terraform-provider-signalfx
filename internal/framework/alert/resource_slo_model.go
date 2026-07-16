// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/slo"

	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type resourceSLOModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Input       types.List   `tfsdk:"input"`
	Target      types.List   `tfsdk:"target"`
}

type sloInputModel struct {
	ProgramText      types.String `tfsdk:"program_text"`
	GoodEventsLabel  types.String `tfsdk:"good_events_label"`
	TotalEventsLabel types.String `tfsdk:"total_events_label"`
}

type sloTargetModel struct {
	Type             types.String  `tfsdk:"type"`
	SLO              types.Float64 `tfsdk:"slo"`
	CompliancePeriod types.String  `tfsdk:"compliance_period"`
	CycleType        types.String  `tfsdk:"cycle_type"`
	CycleStart       types.String  `tfsdk:"cycle_start"`
	AlertRules       types.List    `tfsdk:"alert_rule"`
}

type sloAlertRuleModel struct {
	Type  types.String `tfsdk:"type"`
	Rules types.List   `tfsdk:"rule"`
}

type sloRuleModel struct {
	Severity                    types.String `tfsdk:"severity"`
	Description                 types.String `tfsdk:"description"`
	Notifications               types.List   `tfsdk:"notifications"`
	Disabled                    types.Bool   `tfsdk:"disabled"`
	ParameterizedBody           types.String `tfsdk:"parameterized_body"`
	ParameterizedSubject        types.String `tfsdk:"parameterized_subject"`
	RunbookURL                  types.String `tfsdk:"runbook_url"`
	Tip                         types.String `tfsdk:"tip"`
	SkipClearNotificationStates types.Set    `tfsdk:"skip_clear_notification_states"`
	Parameters                  types.List   `tfsdk:"parameters"`
	ReminderNotification        types.List   `tfsdk:"reminder_notification"`
}

type sloParametersModel struct {
	FireLasting            types.String  `tfsdk:"fire_lasting"`
	PercentOfLasting       types.Float64 `tfsdk:"percent_of_lasting"`
	PercentErrorBudgetLeft types.Float64 `tfsdk:"percent_error_budget_left"`
	ShortWindow1           types.String  `tfsdk:"short_window_1"`
	LongWindow1            types.String  `tfsdk:"long_window_1"`
	ShortWindow2           types.String  `tfsdk:"short_window_2"`
	LongWindow2            types.String  `tfsdk:"long_window_2"`
	BurnRateThreshold1     types.Float64 `tfsdk:"burn_rate_threshold_1"`
	BurnRateThreshold2     types.Float64 `tfsdk:"burn_rate_threshold_2"`
}

func (model resourceSLOModel) request(ctx context.Context) (*slo.SloObject, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	inputs := detectorListElements[sloInputModel](ctx, model.Input, &diagnostics)
	targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
	if len(inputs) != 1 {
		diagnostics.AddError("Invalid SLO input", "Exactly one input block is required.")
	}
	if len(targets) != 1 {
		diagnostics.AddError("Invalid SLO target", "Exactly one target block is required.")
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	target := sloTargetToAPI(ctx, targets[0], &diagnostics)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return &slo.SloObject{
		BaseSlo: slo.BaseSlo{
			Name: model.Name.ValueString(), Description: model.Description.ValueString(),
			Type: model.Type.ValueString(), Targets: []slo.SloTarget{target},
		},
		RequestBasedSlo: &slo.RequestBasedSlo{Inputs: &slo.RequestBasedSloInput{
			ProgramText: inputs[0].ProgramText.ValueString(), GoodEventsLabel: inputs[0].GoodEventsLabel.ValueString(),
			TotalEventsLabel: inputs[0].TotalEventsLabel.ValueString(),
		}},
	}, diagnostics
}

func sloTargetToAPI(ctx context.Context, model sloTargetModel, diagnostics *diag.Diagnostics) slo.SloTarget {
	target := slo.SloTarget{BaseSloTarget: slo.BaseSloTarget{
		Slo: model.SLO.ValueFloat64(), Type: model.Type.ValueString(),
		SloAlertRules: sloAlertRulesToAPI(ctx, model.AlertRules, diagnostics),
	}}
	switch target.Type {
	case slo.RollingWindowTarget:
		target.RollingWindowSloTarget = &slo.RollingWindowSloTarget{CompliancePeriod: model.CompliancePeriod.ValueString()}
	case slo.CalendarWindowTarget:
		target.CalendarWindowSloTarget = &slo.CalendarWindowSloTarget{
			CycleType: model.CycleType.ValueString(), CycleStart: model.CycleStart.ValueString(),
		}
	default:
		diagnostics.AddError("Unsupported SLO target", fmt.Sprintf("Target type %q is not supported.", target.Type))
	}
	return target
}

func sloAlertRulesToAPI(ctx context.Context, value types.List, diagnostics *diag.Diagnostics) []slo.SloAlertRule {
	models := detectorListElements[sloAlertRuleModel](ctx, value, diagnostics)
	result := make([]slo.SloAlertRule, 0, len(models))
	for _, model := range models {
		rules := detectorListElements[sloRuleModel](ctx, model.Rules, diagnostics)
		apiRule := slo.SloAlertRule{BaseSloAlertRule: slo.BaseSloAlertRule{Type: model.Type.ValueString()}}
		switch apiRule.Type {
		case slo.BreachRule:
			values := make([]*slo.BreachDetectorRule, 0, len(rules))
			for _, rule := range rules {
				base, parameters := sloRuleToAPI(ctx, rule, diagnostics)
				value := &slo.BreachDetectorRule{Rule: *base}
				if parameters != nil {
					value.Parameters = &slo.BreachDetectorParameters{
						FireLasting: parameters.FireLasting.ValueString(), PercentOfLasting: parameters.PercentOfLasting.ValueFloat64(),
					}
				}
				values = append(values, value)
			}
			apiRule.BreachSloAlertRule = &slo.BreachSloAlertRule{Rules: values}
		case slo.ErrorBudgetLeftRule:
			values := make([]*slo.ErrorBudgetLeftDetectorRule, 0, len(rules))
			for _, rule := range rules {
				base, parameters := sloRuleToAPI(ctx, rule, diagnostics)
				value := &slo.ErrorBudgetLeftDetectorRule{Rule: *base}
				if parameters != nil {
					value.Parameters = &slo.ErrorBudgetLeftDetectorParameters{
						FireLasting: parameters.FireLasting.ValueString(), PercentOfLasting: parameters.PercentOfLasting.ValueFloat64(),
						PercentErrorBudgetLeft: parameters.PercentErrorBudgetLeft.ValueFloat64(),
					}
				}
				values = append(values, value)
			}
			apiRule.ErrorBudgetLeftSloAlertRule = &slo.ErrorBudgetLeftSloAlertRule{Rules: values}
		case slo.BurnRateRule:
			values := make([]*slo.BurnRateDetectorRule, 0, len(rules))
			for _, rule := range rules {
				base, parameters := sloRuleToAPI(ctx, rule, diagnostics)
				value := &slo.BurnRateDetectorRule{Rule: *base}
				if parameters != nil {
					value.Parameters = &slo.BurnRateDetectorParameters{
						ShortWindow1: parameters.ShortWindow1.ValueString(), LongWindow1: parameters.LongWindow1.ValueString(),
						ShortWindow2: parameters.ShortWindow2.ValueString(), LongWindow2: parameters.LongWindow2.ValueString(),
						BurnRateThreshold1: parameters.BurnRateThreshold1.ValueFloat64(),
						BurnRateThreshold2: parameters.BurnRateThreshold2.ValueFloat64(),
					}
				}
				values = append(values, value)
			}
			apiRule.BurnRateSloAlertRule = &slo.BurnRateSloAlertRule{Rules: values}
		default:
			diagnostics.AddError("Unsupported SLO alert rule", fmt.Sprintf("Alert rule type %q is not supported.", apiRule.Type))
			continue
		}
		result = append(result, apiRule)
	}
	return result
}

func sloRuleToAPI(ctx context.Context, model sloRuleModel, diagnostics *diag.Diagnostics) (*detector.Rule, *sloParametersModel) {
	notifications, notificationDiagnostics := fwshared.NotificationStringsToAPI(ctx, detectorKnownListOrNull(model.Notifications))
	diagnostics.Append(notificationDiagnostics...)
	rule := &detector.Rule{
		Severity: detector.Severity(model.Severity.ValueString()), Description: model.Description.ValueString(),
		Notifications: notifications, Disabled: model.Disabled.ValueBool(), ParameterizedBody: model.ParameterizedBody.ValueString(),
		ParameterizedSubject: model.ParameterizedSubject.ValueString(), RunbookUrl: model.RunbookURL.ValueString(),
		Tip: model.Tip.ValueString(), SkipClearNotificationStates: detectorSetStrings(ctx, model.SkipClearNotificationStates, diagnostics),
	}
	reminders := detectorListElements[detectorReminderNotificationModel](ctx, model.ReminderNotification, diagnostics)
	if len(reminders) > 0 {
		rule.ReminderNotification = &detector.ReminderNotification{
			IntervalMs: reminders[0].IntervalMS.ValueInt64(), TimeoutMs: reminders[0].TimeoutMS.ValueInt64(),
			Type: reminders[0].Type.ValueString(),
		}
	}
	parameters := detectorListElements[sloParametersModel](ctx, model.Parameters, diagnostics)
	if len(parameters) == 0 {
		return rule, nil
	}
	return rule, &parameters[0]
}

func (model *resourceSLOModel) updateFromAPI(ctx context.Context, details *slo.SloObject) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if details == nil {
		diagnostics.AddError("Invalid SLO response", "The SLO API returned no resource data.")
		return diagnostics
	}
	if details.Id == "" {
		diagnostics.AddError("Invalid SLO response", "The SLO API returned no resource identifier.")
		return diagnostics
	}
	if details.Type != slo.RequestBased || details.RequestBasedSlo == nil || details.RequestBasedSlo.Inputs == nil {
		diagnostics.AddError("Invalid SLO response", fmt.Sprintf("The SLO API returned unsupported or incomplete input type %q.", details.Type))
		return diagnostics
	}
	if len(details.Targets) != 1 {
		diagnostics.AddError("Invalid SLO response", fmt.Sprintf("The SLO API returned %d targets; exactly one is required.", len(details.Targets)))
		return diagnostics
	}

	importing := model.Name.IsNull() || model.Name.IsUnknown()
	currentInputs := detectorListElements[sloInputModel](ctx, model.Input, &diagnostics)
	currentTargets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
	var currentInput sloInputModel
	if len(currentInputs) == 1 {
		currentInput = currentInputs[0]
	}
	var currentTarget sloTargetModel
	if len(currentTargets) == 1 {
		currentTarget = currentTargets[0]
	}

	model.ID = types.StringValue(details.Id)
	model.Name = types.StringValue(details.Name)
	model.Description = types.StringValue(details.Description)
	model.Type = types.StringValue(details.Type)
	input := details.RequestBasedSlo.Inputs
	model.Input = detectorListValue(ctx, sloInputAttributeTypes, []sloInputModel{{
		ProgramText:      types.StringValue(input.ProgramText),
		GoodEventsLabel:  sloConfiguredStringFromAPI(currentInput.GoodEventsLabel, input.GoodEventsLabel, importing),
		TotalEventsLabel: sloConfiguredStringFromAPI(currentInput.TotalEventsLabel, input.TotalEventsLabel, importing),
	}}, &diagnostics)
	model.Target = detectorListValue(ctx, sloTargetAttributeTypes, []sloTargetModel{
		sloTargetFromAPI(ctx, currentTarget, details.Targets[0], importing, &diagnostics),
	}, &diagnostics)
	return diagnostics
}

func sloTargetFromAPI(ctx context.Context, current sloTargetModel, target slo.SloTarget, importing bool, diagnostics *diag.Diagnostics) sloTargetModel {
	result := sloTargetModel{
		Type: types.StringValue(target.Type), SLO: types.Float64Value(target.Slo),
		CompliancePeriod: types.StringNull(), CycleType: types.StringNull(), CycleStart: types.StringNull(),
	}
	switch target.Type {
	case slo.RollingWindowTarget:
		if target.RollingWindowSloTarget == nil {
			diagnostics.AddError("Invalid SLO target response", "The rolling-window target payload is missing.")
		} else {
			result.CompliancePeriod = types.StringValue(target.CompliancePeriod)
		}
	case slo.CalendarWindowTarget:
		if target.CalendarWindowSloTarget == nil {
			diagnostics.AddError("Invalid SLO target response", "The calendar-window target payload is missing.")
		} else {
			result.CycleType = types.StringValue(target.CycleType)
			result.CycleStart = types.StringValue(target.CycleStart)
		}
	default:
		diagnostics.AddError("Unsupported SLO target response", fmt.Sprintf("Target type %q is not supported.", target.Type))
	}
	result.AlertRules = sloAlertRulesFromAPI(ctx, current.AlertRules, target.SloAlertRules, importing, diagnostics)
	return result
}

func sloAlertRulesFromAPI(ctx context.Context, currentValue types.List, apiRules []slo.SloAlertRule, importing bool, diagnostics *diag.Diagnostics) types.List {
	current := detectorListElements[sloAlertRuleModel](ctx, currentValue, diagnostics)
	byType := make(map[string]slo.SloAlertRule, len(apiRules))
	for _, rule := range apiRules {
		if _, exists := byType[rule.Type]; exists {
			diagnostics.AddError("Invalid SLO response", fmt.Sprintf("The API returned duplicate alert rule type %q.", rule.Type))
			continue
		}
		byType[rule.Type] = rule
	}
	order := make([]string, 0, len(apiRules))
	if !importing {
		for _, configured := range current {
			if _, ok := byType[configured.Type.ValueString()]; ok {
				order = append(order, configured.Type.ValueString())
			}
		}
	}
	remaining := make([]string, 0, len(apiRules))
	for ruleType := range byType {
		found := false
		for _, existing := range order {
			if existing == ruleType {
				found = true
				break
			}
		}
		if !found {
			remaining = append(remaining, ruleType)
		}
	}
	sort.Strings(remaining)
	order = append(order, remaining...)

	values := make([]sloAlertRuleModel, 0, len(order))
	for _, ruleType := range order {
		var configured sloAlertRuleModel
		for _, item := range current {
			if item.Type.ValueString() == ruleType {
				configured = item
				break
			}
		}
		values = append(values, sloAlertRuleFromAPI(ctx, configured, byType[ruleType], importing, diagnostics))
	}
	if len(values) == 0 {
		diagnostics.AddError("Invalid SLO response", "The SLO API returned no alert rules.")
	}
	return detectorListValue(ctx, sloAlertRuleAttributeTypes, values, diagnostics)
}

func sloAlertRuleFromAPI(ctx context.Context, current sloAlertRuleModel, apiRule slo.SloAlertRule, importing bool, diagnostics *diag.Diagnostics) sloAlertRuleModel {
	currentRules := detectorListElements[sloRuleModel](ctx, current.Rules, diagnostics)
	values := make([]sloRuleModel, 0)
	switch apiRule.Type {
	case slo.BreachRule:
		if apiRule.BreachSloAlertRule == nil {
			diagnostics.AddError("Invalid SLO alert response", "The BREACH rule payload is missing.")
			break
		}
		for index, rule := range apiRule.BreachSloAlertRule.Rules {
			if rule == nil {
				diagnostics.AddError("Invalid SLO alert response", fmt.Sprintf("BREACH rule %d is empty.", index))
				continue
			}
			values = append(values, sloRuleFromAPI(ctx, sloCurrentRule(currentRules, index), &rule.Rule, rule.Parameters, importing, diagnostics))
		}
	case slo.ErrorBudgetLeftRule:
		if apiRule.ErrorBudgetLeftSloAlertRule == nil {
			diagnostics.AddError("Invalid SLO alert response", "The ERROR_BUDGET_LEFT rule payload is missing.")
			break
		}
		for index, rule := range apiRule.ErrorBudgetLeftSloAlertRule.Rules {
			if rule == nil {
				diagnostics.AddError("Invalid SLO alert response", fmt.Sprintf("ERROR_BUDGET_LEFT rule %d is empty.", index))
				continue
			}
			values = append(values, sloRuleFromAPI(ctx, sloCurrentRule(currentRules, index), &rule.Rule, rule.Parameters, importing, diagnostics))
		}
	case slo.BurnRateRule:
		if apiRule.BurnRateSloAlertRule == nil {
			diagnostics.AddError("Invalid SLO alert response", "The BURN_RATE rule payload is missing.")
			break
		}
		for index, rule := range apiRule.BurnRateSloAlertRule.Rules {
			if rule == nil {
				diagnostics.AddError("Invalid SLO alert response", fmt.Sprintf("BURN_RATE rule %d is empty.", index))
				continue
			}
			values = append(values, sloRuleFromAPI(ctx, sloCurrentRule(currentRules, index), &rule.Rule, rule.Parameters, importing, diagnostics))
		}
	default:
		diagnostics.AddError("Unsupported SLO alert response", fmt.Sprintf("Alert rule type %q is not supported.", apiRule.Type))
	}
	if len(values) == 0 {
		diagnostics.AddError("Invalid SLO alert response", fmt.Sprintf("Alert rule type %q contains no rules.", apiRule.Type))
	}
	return sloAlertRuleModel{
		Type:  types.StringValue(apiRule.Type),
		Rules: detectorListValue(ctx, sloRuleAttributeTypes, values, diagnostics),
	}
}

func sloCurrentRule(values []sloRuleModel, index int) sloRuleModel {
	if index < len(values) {
		return values[index]
	}
	return sloRuleModel{}
}

func sloRuleFromAPI(ctx context.Context, current sloRuleModel, rule *detector.Rule, parameters any, importing bool, diagnostics *diag.Diagnostics) sloRuleModel {
	currentNotifications := current.Notifications
	if currentNotifications.ElementType(ctx) == nil {
		currentNotifications = types.ListNull(types.StringType)
	}
	notifications, notificationDiagnostics := fwshared.NotificationStringsFromAPI(ctx, currentNotifications, rule.Notifications)
	diagnostics.Append(notificationDiagnostics...)
	result := sloRuleModel{
		Severity: types.StringValue(string(rule.Severity)), Description: types.StringValue(rule.Description),
		Notifications: notifications, Disabled: types.BoolValue(rule.Disabled),
		ParameterizedBody: types.StringValue(rule.ParameterizedBody), ParameterizedSubject: types.StringValue(rule.ParameterizedSubject),
		RunbookURL: types.StringValue(rule.RunbookUrl), Tip: types.StringValue(rule.Tip),
		SkipClearNotificationStates: detectorStringSetValue(ctx, rule.SkipClearNotificationStates, diagnostics),
		Parameters:                  types.ListNull(types.ObjectType{AttrTypes: sloParametersAttributeTypes}),
		ReminderNotification:        types.ListNull(types.ObjectType{AttrTypes: detectorReminderAttributeTypes}),
	}
	if rule.ReminderNotification != nil && (importing || !current.ReminderNotification.IsNull()) {
		result.ReminderNotification = detectorListValue(ctx, detectorReminderAttributeTypes, []detectorReminderNotificationModel{{
			IntervalMS: types.Int64Value(rule.ReminderNotification.IntervalMs),
			TimeoutMS:  types.Int64Value(rule.ReminderNotification.TimeoutMs), Type: types.StringValue(rule.ReminderNotification.Type),
		}}, diagnostics)
	}
	if parameters != nil && (importing || !current.Parameters.IsNull()) {
		result.Parameters = detectorListValue(ctx, sloParametersAttributeTypes, []sloParametersModel{sloParametersFromAPI(parameters)}, diagnostics)
	}
	return result
}

func sloParametersFromAPI(value any) sloParametersModel {
	result := sloParametersModel{
		FireLasting: types.StringValue(""), PercentOfLasting: types.Float64Value(0),
		PercentErrorBudgetLeft: types.Float64Value(0), ShortWindow1: types.StringValue(""),
		LongWindow1: types.StringValue(""), ShortWindow2: types.StringValue(""), LongWindow2: types.StringValue(""),
		BurnRateThreshold1: types.Float64Value(0), BurnRateThreshold2: types.Float64Value(0),
	}
	switch value := value.(type) {
	case *slo.BreachDetectorParameters:
		result.FireLasting = types.StringValue(value.FireLasting)
		result.PercentOfLasting = types.Float64Value(value.PercentOfLasting)
	case *slo.ErrorBudgetLeftDetectorParameters:
		result.FireLasting = types.StringValue(value.FireLasting)
		result.PercentOfLasting = types.Float64Value(value.PercentOfLasting)
		result.PercentErrorBudgetLeft = types.Float64Value(value.PercentErrorBudgetLeft)
	case *slo.BurnRateDetectorParameters:
		result.ShortWindow1 = types.StringValue(value.ShortWindow1)
		result.LongWindow1 = types.StringValue(value.LongWindow1)
		result.ShortWindow2 = types.StringValue(value.ShortWindow2)
		result.LongWindow2 = types.StringValue(value.LongWindow2)
		result.BurnRateThreshold1 = types.Float64Value(value.BurnRateThreshold1)
		result.BurnRateThreshold2 = types.Float64Value(value.BurnRateThreshold2)
	}
	return result
}

func sloConfiguredStringFromAPI(current types.String, value string, importing bool) types.String {
	if !importing && current.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}

var sloInputAttributeTypes = map[string]attr.Type{
	"program_text": types.StringType, "good_events_label": types.StringType, "total_events_label": types.StringType,
}

var sloParametersAttributeTypes = map[string]attr.Type{
	"fire_lasting": types.StringType, "percent_of_lasting": types.Float64Type,
	"percent_error_budget_left": types.Float64Type, "short_window_1": types.StringType,
	"long_window_1": types.StringType, "short_window_2": types.StringType, "long_window_2": types.StringType,
	"burn_rate_threshold_1": types.Float64Type, "burn_rate_threshold_2": types.Float64Type,
}

var sloRuleAttributeTypes = map[string]attr.Type{
	"severity": types.StringType, "description": types.StringType,
	"notifications": types.ListType{ElemType: types.StringType}, "disabled": types.BoolType,
	"parameterized_body": types.StringType, "parameterized_subject": types.StringType,
	"runbook_url": types.StringType, "tip": types.StringType,
	"skip_clear_notification_states": types.SetType{ElemType: types.StringType},
	"parameters":                     types.ListType{ElemType: types.ObjectType{AttrTypes: sloParametersAttributeTypes}},
	"reminder_notification":          types.ListType{ElemType: types.ObjectType{AttrTypes: detectorReminderAttributeTypes}},
}

var sloAlertRuleAttributeTypes = map[string]attr.Type{
	"type": types.StringType, "rule": types.ListType{ElemType: types.ObjectType{AttrTypes: sloRuleAttributeTypes}},
}

var sloTargetAttributeTypes = map[string]attr.Type{
	"type": types.StringType, "slo": types.Float64Type, "compliance_period": types.StringType,
	"cycle_type": types.StringType, "cycle_start": types.StringType,
	"alert_rule": types.ListType{ElemType: types.ObjectType{AttrTypes: sloAlertRuleAttributeTypes}},
}
