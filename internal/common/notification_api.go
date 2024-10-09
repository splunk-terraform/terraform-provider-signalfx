// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"

	"github.com/signalfx/signalfx-go/notification"
)

func NewStringFromAPI(n *notification.Notification) (string, error) {
	if n == nil {
		return "", fmt.Errorf("nil value provided")
	}
	switch v := n.Value.(type) {
	case *notification.AmazonEventBrigeNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.BigPandaNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.EmailNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.Email), nil
	case *notification.JiraNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.Office365Notification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.OpsgenieNotification:
		return fmt.Sprintf("%s,%s,%s,%s,%s", n.Type, v.CredentialId, v.ResponderName, v.ResponderId, v.ResponderType), nil
	case *notification.PagerDutyNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.ServiceNowNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	case *notification.SlackNotification:
		return fmt.Sprintf("%s,%s,%s", n.Type, v.CredentialId, v.Channel), nil
	case *notification.TeamNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.Team), nil
	case *notification.TeamEmailNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.Team), nil
	case *notification.VictorOpsNotification:
		return fmt.Sprintf("%s,%s,%s", n.Type, v.CredentialId, v.RoutingKey), nil
	case *notification.WebhookNotification:
		return fmt.Sprintf("%s,%s,%s,%s", n.Type, v.CredentialId, v.Secret, v.Url), nil
	case *notification.XMattersNotification:
		return fmt.Sprintf("%s,%s", n.Type, v.CredentialId), nil
	}
	return "", fmt.Errorf("unknown type %T provided", n.Value)
}
