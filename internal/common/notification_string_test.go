// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationFromString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		str    string
		expect *notification.Notification
		errVal string
	}{
		{
			name:   "invalid notification string",
			str:    "invalid",
			expect: nil,
			errVal: "invalid notification string \"invalid\", not enough commas",
		},
		{
			name: "amazon event bridge",
			str:  "AmazonEventBridge,creds",
			expect: &notification.Notification{
				Type: AmazonEventBrigeNotificationType,
				Value: &notification.AmazonEventBrigeNotification{
					Type:         AmazonEventBrigeNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "big panda",
			str:  "BigPanda,creds",
			expect: &notification.Notification{
				Type: BigPandaNotificationType,
				Value: &notification.BigPandaNotification{
					Type:         BigPandaNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name:   "email invalid",
			str:    "Email,invalid",
			expect: nil,
			errVal: "mail: missing '@' or angle-addr",
		},
		{
			name: "email valid",
			str:  "Email,example@localhost",
			expect: &notification.Notification{
				Type: EmailNotificationType,
				Value: &notification.EmailNotification{
					Type:  EmailNotificationType,
					Email: "example@localhost",
				},
			},
			errVal: "",
		},
		{
			name: "jira",
			str:  "Jira,creds",
			expect: &notification.Notification{
				Type: JiraNotificationType,
				Value: &notification.JiraNotification{
					Type:         JiraNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "office 365",
			str:  "Office365,creds",
			expect: &notification.Notification{
				Type: Office365NotificationType,
				Value: &notification.Office365Notification{
					Type:         Office365NotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "opsgenie",
			str:  "Opsgenie,creds,name,id,type",
			expect: &notification.Notification{
				Type: OpsgenieNotificationType,
				Value: &notification.OpsgenieNotification{
					Type:          OpsgenieNotificationType,
					CredentialId:  "creds",
					ResponderName: "name",
					ResponderId:   "id",
					ResponderType: "type",
				},
			},
			errVal: "",
		},
		{
			name:   "opsgenie invalid",
			str:    "Opsgenie,creds,id,type",
			expect: nil,
			errVal: "invalid OpsGenie notification string, please consult the documentation (not enough parts)",
		},
		{
			name: "pager duty",
			str:  "PagerDuty,creds",
			expect: &notification.Notification{
				Type: PagerDutyNotificationType,
				Value: &notification.PagerDutyNotification{
					Type:         PagerDutyNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "service now",
			str:  "ServiceNow,creds",
			expect: &notification.Notification{
				Type: ServiceNowNotificationType,
				Value: &notification.ServiceNowNotification{
					Type:         ServiceNowNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "slack",
			str:  "Slack,creds,channel",
			expect: &notification.Notification{
				Type: SlackNotificationType,
				Value: &notification.SlackNotification{
					Type:         SlackNotificationType,
					CredentialId: "creds",
					Channel:      "channel",
				},
			},
			errVal: "",
		},
		{
			name:   "slack missing channel",
			str:    "Slack,creds",
			expect: nil,
			errVal: "invalid Slack notification string, please consult the documentation (not enough parts)",
		},
		{
			name:   "slack invalid channel",
			str:    "Slack,creds,#channel",
			expect: nil,
			errVal: "exclude the # from channel names in \"#channel\"",
		},
		{
			name: "team",
			str:  "Team,team",
			expect: &notification.Notification{
				Type: TeamNotificationType,
				Value: &notification.TeamNotification{
					Type: TeamNotificationType,
					Team: "team",
				},
			},
			errVal: "",
		},
		{
			name: "team email",
			str:  "TeamEmail,team",
			expect: &notification.Notification{
				Type: TeamEmailNotificationType,
				Value: &notification.TeamEmailNotification{
					Type: TeamEmailNotificationType,
					Team: "team",
				},
			},
			errVal: "",
		},
		{
			name: "victor ops",
			str:  "VictorOps,creds,route",
			expect: &notification.Notification{
				Type: VictorOpsNotificationType,
				Value: &notification.VictorOpsNotification{
					Type:         VictorOpsNotificationType,
					CredentialId: "creds",
					RoutingKey:   "route",
				},
			},
			errVal: "",
		},
		{
			name:   "invalid victor ops",
			str:    "VictorOps,creds",
			expect: nil,
			errVal: "invalid VictorOps notification string, please consult the documentation (not enough parts)",
		},
		{
			name: "webhook creds only",
			str:  "Webhook,creds,,",
			expect: &notification.Notification{
				Type: WebhookNotificationType,
				Value: &notification.WebhookNotification{
					Type:         WebhookNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "webhook secret and url",
			str:  "Webhook,,secret,http://localhost",
			expect: &notification.Notification{
				Type: WebhookNotificationType,
				Value: &notification.WebhookNotification{
					Type:   WebhookNotificationType,
					Secret: "secret",
					Url:    "http://localhost",
				},
			},
			errVal: "",
		},
		{
			name:   "webhook not enough values",
			str:    "Webhook,,",
			expect: nil,
			errVal: "invalid Webhook notification string, please consult the documentation (not enough parts)",
		},
		{
			name:   "webhook no values set",
			str:    "Webhook,,,",
			expect: nil,
			errVal: "invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)",
		},
		{
			name:   "webhook all values set",
			str:    "Webhook,creds,secret,http://localhost",
			expect: nil,
			errVal: "invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)",
		},
		{
			name:   "webhook invalid url",
			str:    "Webhook,,secret,zzz",
			expect: nil,
			errVal: "invalid Webhook URL \"secret\"",
		},
		{
			name: "xmatters",
			str:  "XMatters,creds",
			expect: &notification.Notification{
				Type: XMattersNotificationType,
				Value: &notification.XMattersNotification{
					Type:         XMattersNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name: "splunk platform",
			str:  "SplunkPlatform,creds",
			expect: &notification.Notification{
				Type: SplunkPlatformNotificationType,
				Value: &notification.SplunkPlatformNotification{
					Type:         SplunkPlatformNotificationType,
					CredentialId: "creds",
				},
			},
			errVal: "",
		},
		{
			name:   "invalid provider",
			str:    "invalid,creds",
			expect: nil,
			errVal: "invalid notification type \"invalid\"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := NewNotificationFromString(tc.str)
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
			if tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				require.NoError(t, err, "Must not error when parsing string")
			}
		})
	}
}
