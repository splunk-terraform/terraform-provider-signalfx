// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/detector"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/visual"
)

type resourceDetectorModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	ProgramText           types.String `tfsdk:"program_text"`
	Description           types.String `tfsdk:"description"`
	Timezone              types.String `tfsdk:"timezone"`
	MaxDelay              types.Int64  `tfsdk:"max_delay"`
	MinDelay              types.Int64  `tfsdk:"min_delay"`
	ShowDataMarkers       types.Bool   `tfsdk:"show_data_markers"`
	ShowEventLines        types.Bool   `tfsdk:"show_event_lines"`
	DisableSampling       types.Bool   `tfsdk:"disable_sampling"`
	TimeRange             types.Int64  `tfsdk:"time_range"`
	StartTime             types.Int64  `tfsdk:"start_time"`
	EndTime               types.Int64  `tfsdk:"end_time"`
	Tags                  types.Set    `tfsdk:"tags"`
	Teams                 types.Set    `tfsdk:"teams"`
	Rules                 types.Set    `tfsdk:"rule"`
	AuthorizedWriterTeams types.Set    `tfsdk:"authorized_writer_teams"`
	AuthorizedWriterUsers types.Set    `tfsdk:"authorized_writer_users"`
	VisualizationOptions  types.Set    `tfsdk:"viz_options"`
	LabelResolutions      types.Map    `tfsdk:"label_resolutions"`
	URL                   types.String `tfsdk:"url"`
	DetectorOrigin        types.String `tfsdk:"detector_origin"`
	ParentDetectorID      types.String `tfsdk:"parent_detector_id"`
}

type detectorRuleModel struct {
	Severity                    types.String `tfsdk:"severity"`
	DetectLabel                 types.String `tfsdk:"detect_label"`
	Description                 types.String `tfsdk:"description"`
	Notifications               types.List   `tfsdk:"notifications"`
	Disabled                    types.Bool   `tfsdk:"disabled"`
	ParameterizedBody           types.String `tfsdk:"parameterized_body"`
	ParameterizedSubject        types.String `tfsdk:"parameterized_subject"`
	RunbookURL                  types.String `tfsdk:"runbook_url"`
	Tip                         types.String `tfsdk:"tip"`
	SkipClearNotificationStates types.Set    `tfsdk:"skip_clear_notification_states"`
	ReminderNotification        types.List   `tfsdk:"reminder_notification"`
}

type detectorReminderNotificationModel struct {
	IntervalMS types.Int64  `tfsdk:"interval_ms"`
	TimeoutMS  types.Int64  `tfsdk:"timeout_ms"`
	Type       types.String `tfsdk:"type"`
}

type detectorVisualizationModel struct {
	Label       types.String `tfsdk:"label"`
	Color       types.String `tfsdk:"color"`
	DisplayName types.String `tfsdk:"display_name"`
	ValueUnit   types.String `tfsdk:"value_unit"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
	ValueSuffix types.String `tfsdk:"value_suffix"`
}

func (model resourceDetectorModel) request(ctx context.Context, meta any) (*detector.CreateUpdateDetectorRequest, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	rules := detectorRulesToAPI(ctx, model.Rules, &diagnostics)
	tags := detectorSetStrings(ctx, model.Tags, &diagnostics)
	teams := detectorSetStrings(ctx, model.Teams, &diagnostics)
	authorizedTeams := detectorSetStrings(ctx, model.AuthorizedWriterTeams, &diagnostics)
	authorizedUsers := detectorSetStrings(ctx, model.AuthorizedWriterUsers, &diagnostics)
	visualization := detectorVisualizationToAPI(ctx, model, &diagnostics)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	maxDelay := model.MaxDelay.ValueInt64() * 1000
	minDelay := model.MinDelay.ValueInt64() * 1000
	if maxDelay > detectorMaximumDelayMilliseconds || minDelay > detectorMaximumDelayMilliseconds {
		diagnostics.AddError("Invalid detector delay", "Detector delay values exceed the signed 32-bit millisecond range required by the API.")
		return nil, diagnostics
	}
	maxDelayAPI := int32(maxDelay) // #nosec G115 -- range checked above and constrained by schema.
	minDelayAPI := int32(minDelay) // #nosec G115 -- range checked above and constrained by schema.

	return &detector.CreateUpdateDetectorRequest{
		Name:                 model.Name.ValueString(),
		Description:          model.Description.ValueString(),
		TimeZone:             model.Timezone.ValueString(),
		MaxDelay:             &maxDelayAPI,
		MinDelay:             &minDelayAPI,
		ProgramText:          model.ProgramText.ValueString(),
		Rules:                rules,
		AuthorizedWriters:    &detector.AuthorizedWriters{Teams: authorizedTeams, Users: authorizedUsers},
		Tags:                 common.Unique(pmeta.LoadProviderTags(ctx, meta), tags),
		Teams:                pmeta.MergeProviderTeams(ctx, meta, teams),
		VisualizationOptions: visualization,
		DetectorOrigin:       model.DetectorOrigin.ValueString(),
		ParentDetectorId:     model.ParentDetectorID.ValueString(),
	}, diagnostics
}

func (model resourceDetectorModel) validationRequest(ctx context.Context) (*detector.ValidateDetectorRequestModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	return &detector.ValidateDetectorRequestModel{
		Name:             model.Name.ValueString(),
		ProgramText:      model.ProgramText.ValueString(),
		Rules:            detectorRulesToAPI(ctx, model.Rules, &diagnostics),
		Tags:             detectorSetStrings(ctx, model.Tags, &diagnostics),
		DetectorOrigin:   model.DetectorOrigin.ValueString(),
		ParentDetectorId: model.ParentDetectorID.ValueString(),
	}, diagnostics
}

func detectorRulesToAPI(ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []*detector.Rule {
	models := detectorSetElements[detectorRuleModel](ctx, value, diagnostics)
	rules := make([]*detector.Rule, 0, len(models))
	for _, model := range models {
		apiNotifications, notificationDiagnostics := fwshared.NotificationStringsToAPI(ctx, detectorKnownListOrNull(model.Notifications))
		diagnostics.Append(notificationDiagnostics...)
		states := detectorSetStrings(ctx, model.SkipClearNotificationStates, diagnostics)
		reminders := detectorListElements[detectorReminderNotificationModel](ctx, model.ReminderNotification, diagnostics)
		rule := &detector.Rule{
			Severity:                    detector.Severity(model.Severity.ValueString()),
			DetectLabel:                 model.DetectLabel.ValueString(),
			Description:                 model.Description.ValueString(),
			Notifications:               apiNotifications,
			Disabled:                    model.Disabled.ValueBool(),
			ParameterizedBody:           model.ParameterizedBody.ValueString(),
			ParameterizedSubject:        model.ParameterizedSubject.ValueString(),
			RunbookUrl:                  model.RunbookURL.ValueString(),
			Tip:                         model.Tip.ValueString(),
			SkipClearNotificationStates: states,
		}
		if len(reminders) > 0 {
			rule.ReminderNotification = &detector.ReminderNotification{
				IntervalMs: reminders[0].IntervalMS.ValueInt64(),
				TimeoutMs:  reminders[0].TimeoutMS.ValueInt64(),
				Type:       reminders[0].Type.ValueString(),
			}
		}
		rules = append(rules, rule)
	}
	return rules
}

func detectorVisualizationToAPI(ctx context.Context, model resourceDetectorModel, diagnostics *diag.Diagnostics) *detector.Visualization {
	visualization := &detector.Visualization{
		ShowDataMarkers: model.ShowDataMarkers.ValueBool(),
		ShowEventLines:  model.ShowEventLines.ValueBool(),
		DisableSampling: model.DisableSampling.ValueBool(),
	}
	if !model.StartTime.IsNull() && !model.StartTime.IsUnknown() {
		start := model.StartTime.ValueInt64() * 1000
		visualization.Time = &detector.Time{Type: "absolute", Start: &start}
		if !model.EndTime.IsNull() && !model.EndTime.IsUnknown() {
			end := model.EndTime.ValueInt64() * 1000
			visualization.Time.End = &end
		}
	} else {
		rangeSeconds := int64(3600)
		if !model.TimeRange.IsNull() && !model.TimeRange.IsUnknown() {
			rangeSeconds = model.TimeRange.ValueInt64()
		}
		rangeMilliseconds := rangeSeconds * 1000
		visualization.Time = &detector.Time{Type: "relative", Range: &rangeMilliseconds}
	}

	models := detectorSetElements[detectorVisualizationModel](ctx, model.VisualizationOptions, diagnostics)
	palette := visual.NewColorPalette()
	for _, option := range models {
		apiOption := &detector.PublishLabelOptions{
			Label:       option.Label.ValueString(),
			DisplayName: option.DisplayName.ValueString(),
			ValueUnit:   option.ValueUnit.ValueString(),
			ValuePrefix: option.ValuePrefix.ValueString(),
			ValueSuffix: option.ValueSuffix.ValueString(),
		}
		if option.Color.ValueString() != "" {
			index, ok := palette.ColorIndex(option.Color.ValueString())
			if !ok {
				diagnostics.AddError("Invalid detector visualization color", fmt.Sprintf("Color %q does not exist in the detector palette.", option.Color.ValueString()))
				continue
			}
			apiOption.PaletteIndex = &index
		}
		visualization.PublishLabelOptions = append(visualization.PublishLabelOptions, apiOption)
	}
	return visualization
}

func (model *resourceDetectorModel) updateFromAPI(ctx context.Context, details *detector.Detector, detectorURL string) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if details == nil {
		diagnostics.AddError("Invalid detector response", "The detector API returned no resource data.")
		return diagnostics
	}
	if details.Id == "" {
		diagnostics.AddError("Invalid detector response", "The detector API returned no resource identifier.")
		return diagnostics
	}
	model.ID = types.StringValue(details.Id)
	model.Name = types.StringValue(details.Name)
	model.ProgramText = types.StringValue(details.ProgramText)
	model.Description = types.StringValue(details.Description)
	model.Timezone = detectorResponseString(model.Timezone, details.TimeZone, "UTC")
	model.DetectorOrigin = detectorResponseString(model.DetectorOrigin, details.DetectorOrigin, "Standard")
	model.ParentDetectorID = types.StringValue(details.ParentDetectorId)
	model.MaxDelay = detectorDelayFromAPI(model.MaxDelay, details.MaxDelay)
	model.MinDelay = detectorDelayFromAPI(model.MinDelay, details.MinDelay)
	model.Tags = detectorConfiguredSetOrAPI(ctx, model.Tags, details.Tags, &diagnostics)
	model.Teams = detectorConfiguredSetOrAPI(ctx, model.Teams, details.Teams, &diagnostics)
	model.Rules = detectorRulesFromAPI(ctx, model.Rules, details.Rules, &diagnostics)
	model.LabelResolutions = detectorLabelResolutionsFromAPI(ctx, details.LabelResolutions, &diagnostics)
	model.URL = types.StringValue(detectorURL)

	if details.AuthorizedWriters != nil {
		model.AuthorizedWriterTeams = detectorStringSetValue(ctx, details.AuthorizedWriters.Teams, &diagnostics)
		model.AuthorizedWriterUsers = detectorStringSetValue(ctx, details.AuthorizedWriters.Users, &diagnostics)
	} else {
		model.AuthorizedWriterTeams = detectorKnownSetOrEmpty(model.AuthorizedWriterTeams)
		model.AuthorizedWriterUsers = detectorKnownSetOrEmpty(model.AuthorizedWriterUsers)
	}
	model.updateVisualizationFromAPI(ctx, details.VisualizationOptions, &diagnostics)
	return diagnostics
}

func (model *resourceDetectorModel) updateVisualizationFromAPI(ctx context.Context, visualization *detector.Visualization, diagnostics *diag.Diagnostics) {
	if visualization == nil {
		model.ShowDataMarkers = detectorKnownBoolOr(model.ShowDataMarkers, true)
		model.ShowEventLines = detectorKnownBoolOr(model.ShowEventLines, false)
		model.DisableSampling = detectorKnownBoolOr(model.DisableSampling, false)
		model.TimeRange = detectorKnownInt64Or(model.TimeRange, 3600)
		model.StartTime = detectorKnownInt64OrNull(model.StartTime)
		model.EndTime = detectorKnownInt64OrNull(model.EndTime)
		model.VisualizationOptions = detectorKnownSetOrNull(model.VisualizationOptions, detectorVisualizationAttributeTypes)
		return
	}
	model.ShowDataMarkers = types.BoolValue(visualization.ShowDataMarkers)
	model.ShowEventLines = types.BoolValue(visualization.ShowEventLines)
	model.DisableSampling = types.BoolValue(visualization.DisableSampling)
	switch {
	case visualization.Time != nil && visualization.Time.Range != nil:
		model.TimeRange = types.Int64Value(*visualization.Time.Range / 1000)
		model.StartTime = types.Int64Null()
		model.EndTime = types.Int64Null()
	case visualization.Time != nil && (visualization.Time.Start != nil || visualization.Time.End != nil):
		model.TimeRange = types.Int64Null()
		model.StartTime = detectorOptionalMillisecondsAsSeconds(visualization.Time.Start)
		model.EndTime = detectorOptionalMillisecondsAsSeconds(visualization.Time.End)
	default:
		model.TimeRange = types.Int64Value(3600)
		model.StartTime = types.Int64Null()
		model.EndTime = types.Int64Null()
	}

	current := detectorSetElements[detectorVisualizationModel](ctx, model.VisualizationOptions, diagnostics)
	used := make([]bool, len(current))
	values := make([]detectorVisualizationModel, 0, len(visualization.PublishLabelOptions))
	palette := visual.NewColorPalette()
	for index, option := range visualization.PublishLabelOptions {
		if option == nil {
			diagnostics.AddError("Invalid detector visualization response", fmt.Sprintf("Visualization option at index %d is empty.", index))
			continue
		}
		color := ""
		if option.PaletteIndex != nil {
			var ok bool
			color, ok = palette.IndexColorName(*option.PaletteIndex)
			if !ok {
				diagnostics.AddError("Invalid detector visualization response", fmt.Sprintf("Palette index %d is not supported.", *option.PaletteIndex))
				continue
			}
		}
		configured, matched := detectorMatchVisualization(current, used, option.Label)
		values = append(values, detectorVisualizationModel{
			Label:       types.StringValue(option.Label),
			Color:       detectorNestedStringFromAPI(configured.Color, color, matched),
			DisplayName: detectorNestedStringFromAPI(configured.DisplayName, option.DisplayName, matched),
			ValueUnit:   detectorNestedStringFromAPI(configured.ValueUnit, option.ValueUnit, matched),
			ValuePrefix: detectorNestedStringFromAPI(configured.ValuePrefix, option.ValuePrefix, matched),
			ValueSuffix: detectorNestedStringFromAPI(configured.ValueSuffix, option.ValueSuffix, matched),
		})
	}
	model.VisualizationOptions = detectorSetValue(ctx, detectorVisualizationAttributeTypes, values, diagnostics)
}

func detectorRulesFromAPI(ctx context.Context, currentValue types.Set, rules []*detector.Rule, diagnostics *diag.Diagnostics) types.Set {
	current := detectorSetElements[detectorRuleModel](ctx, currentValue, diagnostics)
	used := make([]bool, len(current))
	values := make([]detectorRuleModel, 0, len(rules))
	for index, rule := range rules {
		if rule == nil {
			diagnostics.AddError("Invalid detector rule response", fmt.Sprintf("Rule at index %d is empty.", index))
			continue
		}
		configured, matched := detectorMatchRule(current, used, rule)
		currentNotifications := types.ListNull(types.StringType)
		if matched {
			currentNotifications = configured.Notifications
		}
		notifications, notificationDiagnostics := fwshared.NotificationStringsFromAPI(ctx, currentNotifications, rule.Notifications)
		diagnostics.Append(notificationDiagnostics...)
		states := detectorNestedStringSetFromAPI(ctx, configured.SkipClearNotificationStates, rule.SkipClearNotificationStates, matched, diagnostics)
		reminders := types.ListNull(types.ObjectType{AttrTypes: detectorReminderAttributeTypes})
		if rule.ReminderNotification != nil {
			configuredReminder, reminderMatched := detectorConfiguredReminder(ctx, configured.ReminderNotification, matched, diagnostics)
			reminders = detectorListValue(ctx, detectorReminderAttributeTypes, []detectorReminderNotificationModel{{
				IntervalMS: types.Int64Value(rule.ReminderNotification.IntervalMs),
				TimeoutMS:  detectorNestedInt64FromAPI(configuredReminder.TimeoutMS, rule.ReminderNotification.TimeoutMs, reminderMatched),
				Type:       types.StringValue(rule.ReminderNotification.Type),
			}}, diagnostics)
		}
		values = append(values, detectorRuleModel{
			Severity:                    types.StringValue(string(rule.Severity)),
			DetectLabel:                 types.StringValue(rule.DetectLabel),
			Description:                 detectorNestedStringFromAPI(configured.Description, rule.Description, matched),
			Notifications:               notifications,
			Disabled:                    detectorNestedBoolFromAPI(configured.Disabled, rule.Disabled, matched),
			ParameterizedBody:           detectorNestedStringFromAPI(configured.ParameterizedBody, rule.ParameterizedBody, matched),
			ParameterizedSubject:        detectorNestedStringFromAPI(configured.ParameterizedSubject, rule.ParameterizedSubject, matched),
			RunbookURL:                  detectorNestedStringFromAPI(configured.RunbookURL, rule.RunbookUrl, matched),
			Tip:                         detectorNestedStringFromAPI(configured.Tip, rule.Tip, matched),
			SkipClearNotificationStates: states,
			ReminderNotification:        reminders,
		})
	}
	if len(values) == 0 {
		diagnostics.AddError("Invalid detector response", "The detector API returned no alert rules.")
	}
	return detectorSetValue(ctx, detectorRuleAttributeTypes, values, diagnostics)
}

func detectorMatchRule(current []detectorRuleModel, used []bool, rule *detector.Rule) (detectorRuleModel, bool) {
	for index, configured := range current {
		if !used[index] && configured.DetectLabel.ValueString() == rule.DetectLabel && configured.Severity.ValueString() == string(rule.Severity) {
			used[index] = true
			return configured, true
		}
	}
	return detectorRuleModel{}, false
}

func detectorMatchVisualization(current []detectorVisualizationModel, used []bool, label string) (detectorVisualizationModel, bool) {
	for index, configured := range current {
		if !used[index] && configured.Label.ValueString() == label {
			used[index] = true
			return configured, true
		}
	}
	return detectorVisualizationModel{}, false
}

func detectorConfiguredReminder(ctx context.Context, value types.List, matched bool, diagnostics *diag.Diagnostics) (detectorReminderNotificationModel, bool) {
	if !matched {
		return detectorReminderNotificationModel{}, false
	}
	values := detectorListElements[detectorReminderNotificationModel](ctx, value, diagnostics)
	if len(values) == 0 {
		return detectorReminderNotificationModel{}, false
	}
	return values[0], true
}

func detectorNestedStringFromAPI(current types.String, value string, matched bool) types.String {
	if matched && current.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func detectorNestedBoolFromAPI(current types.Bool, value bool, matched bool) types.Bool {
	if matched && current.IsNull() {
		return types.BoolNull()
	}
	return types.BoolValue(value)
}

func detectorNestedInt64FromAPI(current types.Int64, value int64, matched bool) types.Int64 {
	if matched && current.IsNull() {
		return types.Int64Null()
	}
	return types.Int64Value(value)
}

func detectorNestedStringSetFromAPI(ctx context.Context, current types.Set, values []string, matched bool, diagnostics *diag.Diagnostics) types.Set {
	if matched && current.IsNull() && len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	return detectorStringSetValue(ctx, values, diagnostics)
}

func detectorLabelResolutionsFromAPI(ctx context.Context, values *map[string]any, diagnostics *diag.Diagnostics) types.Map {
	if values == nil {
		return types.MapNull(types.Int64Type)
	}
	converted := make(map[string]int64, len(*values))
	for key, raw := range *values {
		value, ok := detectorNumericInt64(raw)
		if !ok {
			diagnostics.AddError("Invalid detector label resolution", fmt.Sprintf("Resolution for label %q has unsupported value %v.", key, raw))
			continue
		}
		converted[key] = value
	}
	result, valueDiagnostics := types.MapValueFrom(ctx, types.Int64Type, converted)
	diagnostics.Append(valueDiagnostics...)
	return result
}

func detectorNumericInt64(value any) (int64, bool) {
	switch value := value.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		if value == math.Trunc(value) && value >= math.MinInt64 && value <= math.MaxInt64 {
			return int64(value), true
		}
	case json.Number:
		result, err := value.Int64()
		return result, err == nil
	}
	return 0, false
}

func detectorSetStrings(ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var values []string
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	return values
}

func detectorConfiguredSetOrAPI(ctx context.Context, current types.Set, values []string, diagnostics *diag.Diagnostics) types.Set {
	if !current.IsNull() && !current.IsUnknown() {
		return current
	}
	return detectorStringSetValue(ctx, values, diagnostics)
}

func detectorStringSetValue(ctx context.Context, values []string, diagnostics *diag.Diagnostics) types.Set {
	result, valueDiagnostics := types.SetValueFrom(ctx, types.StringType, values)
	diagnostics.Append(valueDiagnostics...)
	return result
}

func detectorSetElements[T any](ctx context.Context, value types.Set, diagnostics *diag.Diagnostics) []T {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var values []T
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	return values
}

func detectorListElements[T any](ctx context.Context, value types.List, diagnostics *diag.Diagnostics) []T {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var values []T
	diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
	return values
}

func detectorSetValue[T any](ctx context.Context, attributeTypes map[string]attr.Type, values []T, diagnostics *diag.Diagnostics) types.Set {
	result, valueDiagnostics := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, values)
	diagnostics.Append(valueDiagnostics...)
	return result
}

func detectorListValue[T any](ctx context.Context, attributeTypes map[string]attr.Type, values []T, diagnostics *diag.Diagnostics) types.List {
	result, valueDiagnostics := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, values)
	diagnostics.Append(valueDiagnostics...)
	return result
}

func detectorKnownListOrNull(value types.List) types.List {
	if value.IsUnknown() {
		return types.ListNull(types.StringType)
	}
	return value
}

func detectorKnownSetOrEmpty(value types.Set) types.Set {
	if value.IsNull() || value.IsUnknown() {
		return types.SetValueMust(types.StringType, nil)
	}
	return value
}

func detectorKnownSetOrNull(value types.Set, attributeTypes map[string]attr.Type) types.Set {
	if value.IsUnknown() {
		return types.SetNull(types.ObjectType{AttrTypes: attributeTypes})
	}
	return value
}

func detectorKnownBoolOr(value types.Bool, fallback bool) types.Bool {
	if value.IsNull() || value.IsUnknown() {
		return types.BoolValue(fallback)
	}
	return value
}

func detectorKnownInt64Or(value types.Int64, fallback int64) types.Int64 {
	if value.IsNull() || value.IsUnknown() {
		return types.Int64Value(fallback)
	}
	return value
}

func detectorKnownInt64OrNull(value types.Int64) types.Int64 {
	if value.IsUnknown() {
		return types.Int64Null()
	}
	return value
}

func detectorResponseString(current types.String, value, fallback string) types.String {
	if value != "" {
		return types.StringValue(value)
	}
	if !current.IsNull() && !current.IsUnknown() {
		return current
	}
	return types.StringValue(fallback)
}

func detectorDelayFromAPI(current types.Int64, value *int32) types.Int64 {
	if value != nil {
		return types.Int64Value(int64(*value) / 1000)
	}
	return detectorKnownInt64Or(current, 0)
}

func detectorOptionalMillisecondsAsSeconds(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value / 1000)
}

var detectorReminderAttributeTypes = map[string]attr.Type{
	"interval_ms": types.Int64Type, "timeout_ms": types.Int64Type, "type": types.StringType,
}

var detectorRuleAttributeTypes = map[string]attr.Type{
	"severity": types.StringType, "detect_label": types.StringType, "description": types.StringType,
	"notifications": types.ListType{ElemType: types.StringType}, "disabled": types.BoolType,
	"parameterized_body": types.StringType, "parameterized_subject": types.StringType,
	"runbook_url": types.StringType, "tip": types.StringType,
	"skip_clear_notification_states": types.SetType{ElemType: types.StringType},
	"reminder_notification":          types.ListType{ElemType: types.ObjectType{AttrTypes: detectorReminderAttributeTypes}},
}

var detectorVisualizationAttributeTypes = map[string]attr.Type{
	"label": types.StringType, "color": types.StringType, "display_name": types.StringType,
	"value_unit": types.StringType, "value_prefix": types.StringType, "value_suffix": types.StringType,
}
