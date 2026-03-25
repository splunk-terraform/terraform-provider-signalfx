// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/notification"
)

type NotificationModel struct {
	Type           types.String `tfsdk:"type"`
	CredentialID   types.String `tfsdk:"credential_id"`
	Email          types.String `tfsdk:"email"`
	CredentialName types.String `tfsdk:"credential_name"`
	ResponderID    types.String `tfsdk:"responder_id"`
	ResponderName  types.String `tfsdk:"responder_name"`
	ResponderType  types.String `tfsdk:"responder_type"`
	Channel        types.String `tfsdk:"channel"`
	Team           types.String `tfsdk:"team"`
	RoutingKey     types.String `tfsdk:"routing_key"`
	Secret         types.String `tfsdk:"secret"`
	URL            types.String `tfsdk:"url"`
}

func NewNotificationModelFromAPI(n *notification.Notification) NotificationModel {
	if n == nil {
		return NotificationModel{}
	}

	switch notif := n.Value.(type) {
	case *notification.AmazonEventBrigeNotification:
		return NotificationModel{
			Type:         types.StringValue("AmazonEventBridge"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.BigPandaNotification:
		return NotificationModel{
			Type:         types.StringValue("BigPanda"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.EmailNotification:
		return NotificationModel{
			Type:  types.StringValue("Email"),
			Email: types.StringValue(notif.Email),
		}
	case *notification.JiraNotification:
		return NotificationModel{
			Type:         types.StringValue("Jira"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.OpsgenieNotification:
		return NotificationModel{
			Type:          types.StringValue("Opsgenie"),
			CredentialID:  types.StringValue(notif.CredentialId),
			ResponderID:   StringValueOrEmpty(notif.ResponderId),
			ResponderName: StringValueOrEmpty(notif.ResponderName),
			ResponderType: StringValueOrEmpty(notif.ResponderType),
		}
	case *notification.Office365Notification:
		return NotificationModel{
			Type:         types.StringValue("Office365"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.PagerDutyNotification:
		return NotificationModel{
			Type:         types.StringValue("PagerDuty"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.ServiceNowNotification:
		return NotificationModel{
			Type:         types.StringValue("ServiceNow"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	case *notification.SlackNotification:
		return NotificationModel{
			Type:         types.StringValue("Slack"),
			CredentialID: types.StringValue(notif.CredentialId),
			Channel:      types.StringValue(notif.Channel),
		}
	case *notification.TeamEmailNotification:
		return NotificationModel{
			Type: types.StringValue("TeamEmail"),
			Team: types.StringValue(notif.Team),
		}
	case *notification.TeamNotification:
		return NotificationModel{
			Type: types.StringValue("Team"),
			Team: types.StringValue(notif.Team),
		}
	case *notification.VictorOpsNotification:
		return NotificationModel{
			Type:         types.StringValue("VictorOps"),
			CredentialID: types.StringValue(notif.CredentialId),
			RoutingKey:   types.StringValue(notif.RoutingKey),
		}
	case *notification.WebhookNotification:
		return NotificationModel{
			Type:         types.StringValue("Webhook"),
			CredentialID: StringValueOrEmpty(notif.CredentialId),
			Secret:       StringValueOrEmpty(notif.Secret),
			URL:          StringValueOrEmpty(notif.Url),
		}
	case *notification.XMattersNotification:
		return NotificationModel{
			Type:         types.StringValue("XMatters"),
			CredentialID: types.StringValue(notif.CredentialId),
		}
	default:
		return NotificationModel{}
	}
}

func (m *NotificationModel) Notification() *notification.Notification {
	switch t := m.Type.ValueString(); t {
	case "AmazonEventBridge":
		return &notification.Notification{
			Type: t,
			Value: notification.AmazonEventBrigeNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "BigPanda":
		return &notification.Notification{
			Type: t,
			Value: notification.BigPandaNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "Email":
		return &notification.Notification{
			Type: t,
			Value: notification.EmailNotification{
				Type:  t,
				Email: m.Email.ValueString(),
			},
		}
	case "Jira":
		return &notification.Notification{
			Type: t,
			Value: notification.JiraNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "Opsgenie":
		return &notification.Notification{
			Type: t,
			Value: notification.OpsgenieNotification{
				Type:          t,
				CredentialId:  m.CredentialID.ValueString(),
				ResponderId:   m.ResponderID.ValueString(),
				ResponderName: m.ResponderName.ValueString(),
				ResponderType: m.ResponderType.ValueString(),
			},
		}
	case "Office365":
		return &notification.Notification{
			Type: t,
			Value: notification.Office365Notification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "PagerDuty":
		return &notification.Notification{
			Type: t,
			Value: notification.PagerDutyNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "ServiceNow":
		return &notification.Notification{
			Type: t,
			Value: notification.ServiceNowNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	case "Slack":
		return &notification.Notification{
			Type: t,
			Value: notification.SlackNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
				Channel:      m.Channel.ValueString(),
			},
		}
	case "TeamEmail":
		return &notification.Notification{
			Type: t,
			Value: notification.TeamEmailNotification{
				Type: t,
				Team: m.Team.ValueString(),
			},
		}
	case "Team":
		return &notification.Notification{
			Type: t,
			Value: notification.TeamNotification{
				Type: t,
				Team: m.Team.ValueString(),
			},
		}
	case "VictorOps":
		return &notification.Notification{
			Type: t,
			Value: notification.VictorOpsNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
				RoutingKey:   m.RoutingKey.ValueString(),
			},
		}
	case "Webhook":
		return &notification.Notification{
			Type: t,
			Value: notification.WebhookNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
				Secret:       m.Secret.ValueString(),
				Url:          m.URL.ValueString(),
			},
		}
	case "XMatters":
		return &notification.Notification{
			Type: t,
			Value: notification.XMattersNotification{
				Type:         t,
				CredentialId: m.CredentialID.ValueString(),
			},
		}
	}
	return nil
}
