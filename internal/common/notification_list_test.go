// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationList(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		items  []any
		expect []*notification.Notification
		errVal string
	}{
		{
			name:   "no values provided",
			items:  nil,
			expect: nil,
			errVal: "",
		},
		{
			name:   "invalid notification string",
			items:  []any{"Provider"},
			expect: nil,
			errVal: "invalid notification string \"Provider\", not enough commas",
		},
		{
			name:  "valid notification string",
			items: []any{"Email,example@localhost"},
			expect: []*notification.Notification{
				{
					Type: "Email",
					Value: &notification.EmailNotification{
						Type:  "Email",
						Email: "example@localhost",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := NewNotificationList(tc.items)
			assert.Equal(t, tc.expect, actual, "Must match the expected notifications")
			if tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must much the expected error message")
			} else {
				require.NoError(t, err, "Must not error parsing strings")
			}
		})
	}
}

func TestNewNotificationStringList(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		items  []*notification.Notification
		expect []string
		errVal string
	}{
		{
			name:   "nil",
			items:  nil,
			expect: nil,
			errVal: "",
		},
		{
			name:   "empty",
			items:  []*notification.Notification{},
			expect: nil,
			errVal: "",
		},
		{
			name: "valid notification",
			items: []*notification.Notification{
				{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
			},
			expect: []string{"Email,example@com"},
			errVal: "",
		},
		{
			name: "invalid notification",
			items: []*notification.Notification{
				{Type: "Provider", Value: nil},
			},
			expect: nil,
			errVal: "unknown type <nil> provided",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			items, err := NewNotificationStringList(tc.items)
			assert.Equal(t, tc.expect, items, "Must match the expected values")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}
