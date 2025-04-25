// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"reflect"
	"testing"

	"github.com/signalfx/signalfx-go/detector"
)

func TestToReminderNotificationEmptyRules(t *testing.T) {
	rule := map[string]any{}
	result := ToReminderNotification(rule)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestToReminderNotificationValidRule(t *testing.T) {
	rule := map[string]any{
		"reminder_notification": []any{
			map[string]any{
				"interval_ms": 10,
				"timeout_ms":  20,
				"type":        "email",
			},
		},
	}
	expected := &detector.ReminderNotification{
		IntervalMs: 10,
		TimeoutMs:  20,
		Type:       "email",
	}
	result := ToReminderNotification(rule)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToReminderNotificationNilReminder(t *testing.T) {
	rule := map[string]any{
		"reminder_notification": []any{nil},
	}
	result := ToReminderNotification(rule)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestToReminderNotificationMissingFields(t *testing.T) {
	rule := map[string]any{
		"reminder_notification": []any{
			map[string]any{},
		},
	}
	expected := &detector.ReminderNotification{}
	result := ToReminderNotification(rule)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
