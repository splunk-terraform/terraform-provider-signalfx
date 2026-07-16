// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSLODetectorRuleConversion(t *testing.T) {
	t.Parallel()
	input := map[string]any{
		"severity": "Critical", "detect_label": "CPU", "description": "high", "disabled": true,
		"notifications": []any{"Email,alerts@example.com"}, "parameterized_body": "body", "parameterized_subject": "subject",
		"runbook_url": "runbook", "tip": "tip",
		"skip_clear_notification_states": schema.NewSet(schema.HashString, []any{"STOPPED"}),
		"reminder_notification":          []any{map[string]any{"interval_ms": 1000, "timeout_ms": 2000, "type": "TIMEOUT"}},
	}

	rule, err := getDetectorRule(input)
	require.NoError(t, err)
	assert.Equal(t, "Critical", string(rule.Severity))
	assert.Equal(t, "CPU", rule.DetectLabel)
	assert.True(t, rule.Disabled)
	require.Len(t, rule.Notifications, 1)
	require.NotNil(t, rule.ReminderNotification)
	assert.Equal(t, int64(1000), rule.ReminderNotification.IntervalMs)
	assert.Equal(t, []string{"STOPPED"}, rule.SkipClearNotificationStates)

	result, err := getTfDetectorRule(rule)
	require.NoError(t, err)
	assert.Equal(t, "Email,alerts@example.com", result["notifications"].([]string)[0])
	assert.Equal(t, "TIMEOUT", result["reminder_notification"].([]any)[0].(map[string]any)["type"])
}

func TestSLODetectorRuleValidationFailures(t *testing.T) {
	t.Parallel()
	_, errors := validateSeverity("invalid", "severity")
	require.Len(t, errors, 1)
	_, errors = validateSeverity("Info", "severity")
	assert.Empty(t, errors)
	assert.Contains(t, detectorRuleSchema, "reminder_notification")

	_, err := getDetectorRule(map[string]any{
		"severity": "Info", "description": "", "disabled": false, "notifications": []any{"invalid"},
	})
	require.Error(t, err)
}
