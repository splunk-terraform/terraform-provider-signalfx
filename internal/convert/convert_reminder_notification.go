// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"github.com/signalfx/signalfx-go/detector"
)

func ToReminderNotification(tfRule map[string]any) *detector.ReminderNotification {
	if reminders, ok := tfRule["reminder_notification"]; ok && reminders != nil {
		for _, reminder := range reminders.([]any) {
			if reminder != nil {
				reminder := reminder.(map[string]any)
				reminderNotification := &detector.ReminderNotification{}
				if interval, ok := reminder["interval_ms"]; ok {
					reminderNotification.IntervalMs = int64(interval.(int))
				}
				if timeout, ok := reminder["timeout_ms"]; ok {
					reminderNotification.TimeoutMs = int64(timeout.(int))
				}
				if reminderType, ok := reminder["type"]; ok {
					reminderNotification.Type = reminderType.(string)
				}
				return reminderNotification
			}
		}
	}
	return nil
}
