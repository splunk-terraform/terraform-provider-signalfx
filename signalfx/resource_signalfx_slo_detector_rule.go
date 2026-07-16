// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
)

// These SDKv2 rule helpers are owned by the remaining SLO resource. They are
// removed with the SLO Framework migration rather than retained by detector.
var detectorRuleSchema = map[string]*schema.Schema{
	"severity": {
		Type: schema.TypeString, Required: true, ValidateFunc: validateSeverity,
		Description: "The severity of the rule, must be one of: Critical, Warning, Major, Minor, Info",
	},
	"detect_label": {Type: schema.TypeString, Required: true, Description: "A detect label which matches a detect label within the program text"},
	"description":  {Type: schema.TypeString, Optional: true, Description: "Description of the rule"},
	"notifications": {
		Type: schema.TypeList, Optional: true,
		Elem:        &schema.Schema{Type: schema.TypeString, ValidateDiagFunc: check.Notification()},
		Description: "List of strings specifying where notifications will be sent when an incident occurs",
	},
	"disabled":              {Type: schema.TypeBool, Optional: true, Default: false, Description: "When true, notifications and events will not be generated for the rule"},
	"parameterized_body":    {Type: schema.TypeString, Optional: true, Description: "Custom notification message body when an alert is triggered"},
	"parameterized_subject": {Type: schema.TypeString, Optional: true, Description: "Custom notification message subject when an alert is triggered"},
	"runbook_url":           {Type: schema.TypeString, Optional: true, Description: "URL of page to consult when an alert is triggered"},
	"tip":                   {Type: schema.TypeString, Optional: true, Description: "Plain text suggested first course of action"},
	"skip_clear_notification_states": {
		Type: schema.TypeSet, Optional: true,
		Elem:        &schema.Schema{Type: schema.TypeString, ValidateDiagFunc: check.AlertClearState()},
		Description: "Alert clear states for which clear notifications are not sent",
	},
	"reminder_notification": {
		Type: schema.TypeList, Optional: true, MaxItems: 1,
		Description: "Repeated notification settings for an active alert",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"interval_ms": {Type: schema.TypeInt, Required: true, Description: "Notification interval in milliseconds"},
			"timeout_ms":  {Type: schema.TypeInt, Optional: true, Description: "Notification timeout in milliseconds"},
			"type":        {Type: schema.TypeString, Required: true, ValidateDiagFunc: check.NotificationReminderType(), Description: "Reminder type"},
		}},
	},
}

func getDetectorRule(tfRule map[string]any) (*detector.Rule, error) {
	rule := &detector.Rule{Description: tfRule["description"].(string), Disabled: tfRule["disabled"].(bool)}
	if detectLabel, ok := tfRule["detect_label"]; ok {
		rule.DetectLabel = detectLabel.(string)
	}
	switch tfRule["severity"].(string) {
	case "Critical":
		rule.Severity = detector.CRITICAL
	case "Warning":
		rule.Severity = detector.WARNING
	case "Major":
		rule.Severity = detector.MAJOR
	case "Minor":
		rule.Severity = detector.MINOR
	default:
		rule.Severity = detector.INFO
	}
	if value, ok := tfRule["parameterized_body"]; ok {
		rule.ParameterizedBody = value.(string)
	}
	if value, ok := tfRule["parameterized_subject"]; ok {
		rule.ParameterizedSubject = value.(string)
	}
	if value, ok := tfRule["runbook_url"]; ok {
		rule.RunbookUrl = value.(string)
	}
	if value, ok := tfRule["tip"]; ok {
		rule.Tip = value.(string)
	}
	if notifications, ok := tfRule["notifications"]; ok {
		values, err := common.NewNotificationList(notifications.([]any))
		if err != nil {
			return nil, err
		}
		rule.Notifications = values
	}
	rule.ReminderNotification = convert.ToReminderNotification(tfRule)
	if states, ok := tfRule["skip_clear_notification_states"].(*schema.Set); ok {
		for _, state := range states.List() {
			rule.SkipClearNotificationStates = append(rule.SkipClearNotificationStates, state.(string))
		}
	}
	return rule, nil
}

func getTfDetectorRule(rule *detector.Rule) (map[string]any, error) {
	notifications := make([]string, len(rule.Notifications))
	for index, item := range rule.Notifications {
		value, err := common.NewNotificationStringFromAPI(item)
		if err != nil {
			return nil, err
		}
		notifications[index] = value
	}
	result := map[string]any{
		"severity": rule.Severity, "detect_label": rule.DetectLabel, "description": rule.Description,
		"notifications": notifications, "disabled": rule.Disabled, "parameterized_body": rule.ParameterizedBody,
		"parameterized_subject": rule.ParameterizedSubject, "runbook_url": rule.RunbookUrl, "tip": rule.Tip,
		"skip_clear_notification_states": rule.SkipClearNotificationStates,
	}
	if rule.ReminderNotification != nil {
		result["reminder_notification"] = []any{map[string]any{
			"interval_ms": rule.ReminderNotification.IntervalMs,
			"timeout_ms":  rule.ReminderNotification.TimeoutMs,
			"type":        rule.ReminderNotification.Type,
		}}
	}
	return result, nil
}

func validateSeverity(value any, _ string) (warnings []string, errors []error) {
	severity := value.(string)
	for _, allowed := range []string{"Critical", "Major", "Minor", "Warning", "Info"} {
		if severity == allowed {
			return nil, nil
		}
	}
	return nil, []error{fmt.Errorf("%s not allowed; must be one of: %s", severity, strings.Join([]string{"Critical", "Major", "Minor", "Warning", "Info"}, ", "))}
}
