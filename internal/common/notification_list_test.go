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
		tc := tc
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
