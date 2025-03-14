// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package orgtoken

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/orgtoken"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

func TestSchemaDecode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values map[string]any
		expect *orgtoken.Token
		errVal string
	}{
		{
			name:   "no values provided",
			values: map[string]any{},
			expect: &orgtoken.Token{
				Limits: &orgtoken.Limit{},
			},
			errVal: "",
		},
		{
			name: "dpm limits set",
			values: map[string]any{
				"name":        "my awesome token",
				"description": "This is a test token",
				"secret":      "derp",
				"disabled":    false,
				"expires_at":  int(time.Unix(0, 10000).UnixMilli()),
				"dpm_limits": []any{
					map[string]any{"dpm_limit": 1000, "dpm_notification_threshold": 2000},
				},
				"notifications": []any{"Email,example@com"},
			},
			expect: &orgtoken.Token{
				Name:        "my awesome token",
				Description: "This is a test token",
				Secret:      "derp",
				Disabled:    false,
				Expiry:      time.Unix(0, 10000).UnixMilli(),
				Limits: &orgtoken.Limit{
					DpmQuota:                 common.AsPointer[int32](1000),
					DpmNotificationThreshold: common.AsPointer[int32](2000),
				},
				Notifications: []*notification.Notification{
					{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
				},
			},
		},
		{
			name: "mts limits set",
			values: map[string]any{
				"name":        "my awesome token",
				"description": "this is a test token",
				"secret":      "aabb",
				"disabled":    true,
				"expires_at":  int(time.Unix(0, 100).UnixMilli()),
				"auth_scopes": []any{"power"},
				"host_or_usage_limits": []any{
					map[string]any{
						"host_notification_threshold":             100,
						"host_limit":                              1000,
						"container_notification_threshold":        1000,
						"container_limit":                         10,
						"custom_metrics_notification_threshold":   1,
						"custom_metrics_limit":                    1,
						"high_res_metrics_notification_threshold": 1,
						"high_res_metrics_limit":                  1,
					},
				},
			},
			expect: &orgtoken.Token{
				Name:        "my awesome token",
				Description: "this is a test token",
				Secret:      "aabb",
				Disabled:    true,
				Expiry:      time.Unix(0, 100).UnixMilli(),
				AuthScopes:  []string{"power"},
				Limits: &orgtoken.Limit{
					CategoryQuota: &orgtoken.UsageLimits{
						HostThreshold:          common.AsPointer[int64](1000),
						ContainerThreshold:     common.AsPointer[int64](10),
						CustomMetricThreshold:  common.AsPointer[int64](1),
						HighResMetricThreshold: common.AsPointer[int64](1),
					},
					CategoryNotificationThreshold: &orgtoken.UsageLimits{
						HostThreshold:          common.AsPointer[int64](100),
						ContainerThreshold:     common.AsPointer[int64](1000),
						CustomMetricThreshold:  common.AsPointer[int64](1),
						HighResMetricThreshold: common.AsPointer[int64](1),
					},
				},
			},
			errVal: "",
		},
		{
			name: "invalid notification",
			values: map[string]any{
				"notifications": []any{0},
			},
			expect: nil,
			errVal: "invalid notification string \"0\", not enough commas",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := schema.TestResourceDataRaw(t, newSchema(), tc.values)

			token, err := decodeTerraform(data)
			assert.Equal(t, tc.expect, token, "Must match the expected token")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error encode")
			}
		})
	}
}

func TestSchemaEncode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		token  *orgtoken.Token
		errVal string
	}{
		{
			name:   "empty token",
			token:  &orgtoken.Token{},
			errVal: "",
		},
		{
			name:   "broken notifications",
			token:  &orgtoken.Token{Notifications: []*notification.Notification{{Type: "broken"}}},
			errVal: "notifications: unknown type <nil> provided",
		},
		{
			name: "dpm based token",
			token: &orgtoken.Token{
				Name: "my awesome token",
				Limits: &orgtoken.Limit{
					DpmQuota:                 common.AsPointer[int32](200),
					DpmNotificationThreshold: common.AsPointer[int32](100),
				},
			},
			errVal: "",
		},
		{
			name: "mts based token",
			token: &orgtoken.Token{
				Name: "my awesome token",
				Limits: &orgtoken.Limit{
					CategoryQuota: &orgtoken.UsageLimits{
						HostThreshold: common.AsPointer[int64](1),
					},
					CategoryNotificationThreshold: &orgtoken.UsageLimits{
						HostThreshold: common.AsPointer[int64](1),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := schema.TestResourceDataRaw(t, newSchema(), map[string]any{})

			err := encodeTerraform(tc.token, data)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}
