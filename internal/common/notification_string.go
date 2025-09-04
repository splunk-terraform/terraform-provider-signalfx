// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/signalfx/signalfx-go/notification"
)

const (
	AmazonEventBrigeNotificationType string = "AmazonEventBridge"
	BigPandaNotificationType         string = "BigPanda"
	EmailNotificationType            string = "Email"
	JiraNotificationType             string = "Jira"
	Office365NotificationType        string = "Office365"
	OpsgenieNotificationType         string = "Opsgenie"
	PagerDutyNotificationType        string = "PagerDuty"
	ServiceNowNotificationType       string = "ServiceNow"
	SlackNotificationType            string = "Slack"
	SplunkPlatformNotificationType   string = "SplunkPlatform"
	TeamNotificationType             string = "Team"
	TeamEmailNotificationType        string = "TeamEmail"
	VictorOpsNotificationType        string = "VictorOps"
	WebhookNotificationType          string = "Webhook"
	XMattersNotificationType         string = "XMatters"
)

// NewNotificationFromString converts the notification definition into the API type.
func NewNotificationFromString(str string) (*notification.Notification, error) {
	var (
		values = strings.Split(str, ",")
		count  = len(values)
		value  any
	)
	if count < 2 {
		return nil, fmt.Errorf("invalid notification string %q, not enough commas", str)
	}
	switch values[0] {
	case AmazonEventBrigeNotificationType:
		value = &notification.AmazonEventBrigeNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case BigPandaNotificationType:
		value = &notification.BigPandaNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case EmailNotificationType:
		if _, err := mail.ParseAddress(values[1]); err != nil {
			return nil, err
		}
		value = &notification.EmailNotification{
			Type:  values[0],
			Email: values[1],
		}
	case JiraNotificationType:
		value = &notification.JiraNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case Office365NotificationType:
		value = &notification.Office365Notification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case OpsgenieNotificationType:
		if count != 5 {
			return nil, fmt.Errorf("invalid OpsGenie notification string, please consult the documentation (not enough parts)")
		}
		value = &notification.OpsgenieNotification{
			Type:          values[0],
			CredentialId:  values[1],
			ResponderName: values[2],
			ResponderId:   values[3],
			ResponderType: values[4],
		}
	case PagerDutyNotificationType:
		value = &notification.PagerDutyNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case ServiceNowNotificationType:
		value = &notification.ServiceNowNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case SlackNotificationType:
		if count != 3 {
			return nil, fmt.Errorf("invalid Slack notification string, please consult the documentation (not enough parts)")
		}
		if strings.Contains(values[2], "#") {
			return nil, fmt.Errorf("exclude the # from channel names in %q", values[2])
		}
		value = &notification.SlackNotification{
			Type:         values[0],
			CredentialId: values[1],
			Channel:      values[2],
		}
	case SplunkPlatformNotificationType:
		value = &notification.SplunkPlatformNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	case TeamNotificationType:
		value = &notification.TeamNotification{
			Type: values[0],
			Team: values[1],
		}
	case TeamEmailNotificationType:
		value = &notification.TeamEmailNotification{
			Type: values[0],
			Team: values[1],
		}
	case VictorOpsNotificationType:
		if count != 3 {
			return nil, fmt.Errorf("invalid VictorOps notification string, please consult the documentation (not enough parts)")
		}
		value = &notification.VictorOpsNotification{
			Type:         values[0],
			CredentialId: values[1],
			RoutingKey:   values[2],
		}
	case WebhookNotificationType:
		if count != 4 {
			return nil, fmt.Errorf("invalid Webhook notification string, please consult the documentation (not enough parts)")
		}
		if values[1] != "" {
			// We got a credential ID, so verify we didn't get the other parts
			if values[2] != "" || values[3] != "" {
				return nil, fmt.Errorf("invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)")
			}
		} else {
			// We didn't get a credential ID so verify we got the other parts
			if values[2] == "" || values[3] == "" {
				return nil, fmt.Errorf("invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)")
			}
		}
		// The URL might be an empty string in the case that the user
		// only supplied a credential ID
		if values[3] != "" {
			if _, err := url.ParseRequestURI(values[3]); err != nil {
				return nil, fmt.Errorf("invalid Webhook URL %q", values[2])
			}
		}
		value = &notification.WebhookNotification{
			Type:         values[0],
			CredentialId: values[1],
			Secret:       values[2],
			Url:          values[3],
		}
	case XMattersNotificationType:
		value = &notification.XMattersNotification{
			Type:         values[0],
			CredentialId: values[1],
		}
	default:
		return nil, fmt.Errorf("invalid notification type %q", values[0])
	}

	return &notification.Notification{Type: values[0], Value: value}, nil
}
