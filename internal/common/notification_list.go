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
