// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"github.com/signalfx/signalfx-go/detector"
)

func ToReminderNotification(tfRule map[string]interface{}) *detector.ReminderNotification {
	if reminders, ok := tfRule["reminder_notification"]; ok && reminders != nil {
		for _, reminder := range reminders.([]interface{}) {
			if reminder != nil {
				reminder := reminder.(map[string]interface{})
				reminderNotification := &detector.ReminderNotification{}
				if interval, ok := reminder["interval"]; ok {
					reminderNotification.Interval = int64(interval.(int))
				}
				if timeout, ok := reminder["timeout"]; ok {
					reminderNotification.Timeout = int64(timeout.(int))
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
