// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"github.com/signalfx/signalfx-go/notification"
)

func NewNotificationList(items []any) ([]*notification.Notification, error) {
	if len(items) == 0 {
		return nil, nil
	}
	values := make([]*notification.Notification, len(items))
	for i, v := range items {
		n, err := NewNotificationFromString(v.(string))
		if err != nil {
			return nil, err
		}
		values[i] = n
	}
	return values, nil
}

func NewNotificationStringList(items []*notification.Notification) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	values := make([]string, len(items))
	for i, v := range items {
		var err error
		values[i], err = NewNotificationStringFromAPI(v)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
