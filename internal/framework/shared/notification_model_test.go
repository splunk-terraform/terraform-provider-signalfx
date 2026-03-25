// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0
package fwshared

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
)

func TestNotificationModel(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		model    NotificationModel
		expected *notification.Notification
	}{
		{
			name: "AmazonEventBridge",
			model: NotificationModel{
				Type:         types.StringValue("AmazonEventBridge"),
				CredentialID: types.StringValue("cred-123"),
			},
			expected: &notification.Notification{
				Type: "AmazonEventBridge",
				Value: notification.AmazonEventBrigeNotification{
					Type:         "AmazonEventBridge",
					CredentialId: "cred-123",
				},
			},
		},
		{
			name: "BigPanda",
			model: NotificationModel{
				Type:         types.StringValue("BigPanda"),
				CredentialID: types.StringValue("cred-234"),
			},
			expected: &notification.Notification{
				Type: "BigPanda",
				Value: notification.BigPandaNotification{
					Type:         "BigPanda",
					CredentialId: "cred-234",
				},
			},
		},
		{
			name: "Email",
			model: NotificationModel{
				Type:  types.StringValue("Email"),
				Email: types.StringValue("test@example.com"),
			},
			expected: &notification.Notification{
				Type: "Email",
				Value: notification.EmailNotification{
					Type:  "Email",
					Email: "test@example.com",
				},
			},
		},
		{
			name: "Jira",
			model: NotificationModel{
				Type:         types.StringValue("Jira"),
				CredentialID: types.StringValue("cred-345"),
			},
			expected: &notification.Notification{
				Type: "Jira",
				Value: notification.JiraNotification{
					Type:         "Jira",
					CredentialId: "cred-345",
				},
			},
		},
		{
			name: "Opsgenie",
			model: NotificationModel{
				Type:          types.StringValue("Opsgenie"),
				CredentialID:  types.StringValue("cred-456"),
				ResponderID:   types.StringValue("responder-1"),
				ResponderName: types.StringValue("Team A"),
				ResponderType: types.StringValue("team"),
			},
			expected: &notification.Notification{
				Type: "Opsgenie",
				Value: notification.OpsgenieNotification{
					Type:          "Opsgenie",
					CredentialId:  "cred-456",
					ResponderId:   "responder-1",
					ResponderName: "Team A",
					ResponderType: "team",
				},
			},
		},
		{
			name: "Office365",
			model: NotificationModel{
				Type:         types.StringValue("Office365"),
				CredentialID: types.StringValue("cred-567"),
			},
			expected: &notification.Notification{
				Type: "Office365",
				Value: notification.Office365Notification{
					Type:         "Office365",
					CredentialId: "cred-567",
				},
			},
		},
		{
			name: "PagerDuty",
			model: NotificationModel{
				Type:         types.StringValue("PagerDuty"),
				CredentialID: types.StringValue("cred-678"),
			},
			expected: &notification.Notification{
				Type: "PagerDuty",
				Value: notification.PagerDutyNotification{
					Type:         "PagerDuty",
					CredentialId: "cred-678",
				},
			},
		},
		{
			name: "ServiceNow",
			model: NotificationModel{
				Type:         types.StringValue("ServiceNow"),
				CredentialID: types.StringValue("cred-789"),
			},
			expected: &notification.Notification{
				Type: "ServiceNow",
				Value: notification.ServiceNowNotification{
					Type:         "ServiceNow",
					CredentialId: "cred-789",
				},
			},
		},
		{
			name: "Slack",
			model: NotificationModel{
				Type:         types.StringValue("Slack"),
				CredentialID: types.StringValue("cred-890"),
				Channel:      types.StringValue("#alerts"),
			},
			expected: &notification.Notification{
				Type: "Slack",
				Value: notification.SlackNotification{
					Type:         "Slack",
					CredentialId: "cred-890",
					Channel:      "#alerts",
				},
			},
		},
		{
			name: "TeamEmail",
			model: NotificationModel{
				Type: types.StringValue("TeamEmail"),
				Team: types.StringValue("platform"),
			},
			expected: &notification.Notification{
				Type: "TeamEmail",
				Value: notification.TeamEmailNotification{
					Type: "TeamEmail",
					Team: "platform",
				},
			},
		},
		{
			name: "Team",
			model: NotificationModel{
				Type: types.StringValue("Team"),
				Team: types.StringValue("platform"),
			},
			expected: &notification.Notification{
				Type: "Team",
				Value: notification.TeamNotification{
					Type: "Team",
					Team: "platform",
				},
			},
		},
		{
			name: "VictorOps",
			model: NotificationModel{
				Type:         types.StringValue("VictorOps"),
				CredentialID: types.StringValue("cred-901"),
				RoutingKey:   types.StringValue("route-key"),
			},
			expected: &notification.Notification{
				Type: "VictorOps",
				Value: notification.VictorOpsNotification{
					Type:         "VictorOps",
					CredentialId: "cred-901",
					RoutingKey:   "route-key",
				},
			},
		},
		{
			name: "Webhook",
			model: NotificationModel{
				Type:         types.StringValue("Webhook"),
				CredentialID: types.StringValue("cred-012"),
				Secret:       types.StringValue("secret-value"),
				URL:          types.StringValue("https://example.com/webhook"),
			},
			expected: &notification.Notification{
				Type: "Webhook",
				Value: notification.WebhookNotification{
					Type:         "Webhook",
					CredentialId: "cred-012",
					Secret:       "secret-value",
					Url:          "https://example.com/webhook",
				},
			},
		},
		{
			name: "XMatters",
			model: NotificationModel{
				Type:         types.StringValue("XMatters"),
				CredentialID: types.StringValue("cred-1234"),
			},
			expected: &notification.Notification{
				Type: "XMatters",
				Value: notification.XMattersNotification{
					Type:         "XMatters",
					CredentialId: "cred-1234",
				},
			},
		},
		{
			name: "unknown type",
			model: NotificationModel{
				Type: types.StringValue("Unknown"),
			},
			expected: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.model.Notification()
			assert.Equal(t, tt.expected, result)
		})
	}
}
