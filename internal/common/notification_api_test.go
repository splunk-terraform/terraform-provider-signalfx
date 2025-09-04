// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/require"
)

func TestNewStringFromAPI(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		nt     *notification.Notification
		expect string
		errVal string
	}{
		{
			name:   "no value set",
			nt:     nil,
			expect: "",
			errVal: "nil value provided",
		},
		{
			name: "invalid type set",
			nt: &notification.Notification{
				Type:  "brrr",
				Value: nil,
			},
			expect: "",
			errVal: "unknown type <nil> provided",
		},
		{
			name: "amazon event bridge",
			nt: &notification.Notification{
				Type: AmazonEventBrigeNotificationType,
				Value: &notification.AmazonEventBrigeNotification{
					Type:         AmazonEventBrigeNotificationType,
					CredentialId: "aaa",
				},
			},
			expect: "AmazonEventBridge,aaa",
			errVal: "",
		},
		{
			name: "big panda",
			nt: &notification.Notification{
				Type: BigPandaNotificationType,
				Value: &notification.BigPandaNotification{
					Type:         BigPandaNotificationType,
					CredentialId: "bbb",
				},
			},
			expect: "BigPanda,bbb",
			errVal: "",
		},
		{
			name: "email",
			nt: &notification.Notification{
				Type: EmailNotificationType,
				Value: &notification.EmailNotification{
					Type:  EmailNotificationType,
					Email: "example@localhost",
				},
			},
			expect: "Email,example@localhost",
			errVal: "",
		},
		{
			name: "jira",
			nt: &notification.Notification{
				Type: JiraNotificationType,
				Value: &notification.JiraNotification{
					Type:         JiraNotificationType,
					CredentialId: "ccc",
				},
			},
			expect: "Jira,ccc",
			errVal: "",
		},
		{
			name: "office 365",
			nt: &notification.Notification{
				Type: Office365NotificationType,
				Value: &notification.Office365Notification{
					Type:         Office365NotificationType,
					CredentialId: "ddd",
				},
			},
			expect: "Office365,ddd",
			errVal: "",
		},
		{
			name: "opsgenie",
			nt: &notification.Notification{
				Type: OpsgenieNotificationType,
				Value: &notification.OpsgenieNotification{
					Type:          OpsgenieNotificationType,
					CredentialId:  "eee",
					ResponderName: "john",
					ResponderId:   "id",
					ResponderType: "human",
				},
			},
			expect: "Opsgenie,eee,john,id,human",
			errVal: "",
		},
		{
			name: "pager duty",
			nt: &notification.Notification{
				Type: PagerDutyNotificationType,
				Value: &notification.PagerDutyNotification{
					Type:         PagerDutyNotificationType,
					CredentialId: "fff",
				},
			},
			expect: "PagerDuty,fff",
			errVal: "",
		},
		{
			name: "service now",
			nt: &notification.Notification{
				Type: ServiceNowNotificationType,
				Value: &notification.ServiceNowNotification{
					Type:         ServiceNowNotificationType,
					CredentialId: "ggg",
				},
			},
			expect: "ServiceNow,ggg",
			errVal: "",
		},
		{
			name: "slack",
			nt: &notification.Notification{
				Type: SlackNotificationType,
				Value: &notification.SlackNotification{
					Type:         SlackNotificationType,
					CredentialId: "hhh",
					Channel:      "announcements",
				},
			},
			expect: "Slack,hhh,announcements",
			errVal: "",
		},
		{
			name: "splunk platform",
			nt: &notification.Notification{
				Type: SplunkPlatformNotificationType,
				Value: &notification.SplunkPlatformNotification{
					Type:         SplunkPlatformNotificationType,
					CredentialId: "iii",
				},
			},
			expect: "SplunkPlatform,iii",
			errVal: "",
		},
		{
			name: "team",
			nt: &notification.Notification{
				Type: TeamNotificationType,
				Value: &notification.TeamNotification{
					Type: TeamNotificationType,
					Team: "default",
				},
			},
			expect: "Team,default",
			errVal: "",
		},
		{
			name: "team email",
			nt: &notification.Notification{
				Type: TeamEmailNotificationType,
				Value: &notification.TeamEmailNotification{
					Type: TeamEmailNotificationType,
					Team: "default",
				},
			},
			expect: "TeamEmail,default",
			errVal: "",
		},
		{
			name: "victor ops",
			nt: &notification.Notification{
				Type: VictorOpsNotificationType,
				Value: &notification.VictorOpsNotification{
					Type:         VictorOpsNotificationType,
					CredentialId: "iii",
					RoutingKey:   "sre",
				},
			},
			expect: "VictorOps,iii,sre",
			errVal: "",
		},
		{
			name: "webhook",
			nt: &notification.Notification{
				Type: WebhookNotificationType,
				Value: &notification.WebhookNotification{
					Type:         WebhookNotificationType,
					CredentialId: "jjj",
					Secret:       "hunter2",
					Url:          "http://localhost",
				},
			},
			expect: "Webhook,jjj,hunter2,http://localhost",
			errVal: "",
		},
		{
			name: "x matters",
			nt: &notification.Notification{
				Type: XMattersNotificationType,
				Value: &notification.XMattersNotification{
					Type:         XMattersNotificationType,
					CredentialId: "lll",
				},
			},
			expect: "XMatters,lll",
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := NewNotificationStringFromAPI(tc.nt)
			require.Equal(t, tc.expect, actual, "Must match the expected error string")
			if tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must match the expected error message")
			} else {
				require.NoError(t, err, "Must not error ")
			}
		})
	}
}
