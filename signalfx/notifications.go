package signalfx

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/signalfx/signalfx-go/notification"
)

const (
	BigPandaNotificationType   string = "BigPanda"
	EmailNotificationType      string = "Email"
	JiraNotificationType       string = "Jira"
	Office365NotificationType  string = "Office365"
	OpsgenieNotificationType   string = "Opsgenie"
	PagerDutyNotificationType  string = "PagerDuty"
	ServiceNowNotificationType string = "ServiceNow"
	SlackNotificationType      string = "Slack"
	TeamNotificationType       string = "Team"
	TeamEmailNotificationType  string = "TeamEmail"
	VictorOpsNotificationType  string = "VictorOps"
	WebhookNotificationType    string = "Webhook"
	XMattersNotificationType   string = "XMatters"
)

func getNotifyStringFromAPI(not *notification.Notification) (string, error) {
	nt := not.Type
	switch nt {
	case BigPandaNotificationType:
		bp := not.Value.(*notification.BigPandaNotification)
		return fmt.Sprintf("%s,%s", nt, bp.CredentialId), nil
	case EmailNotificationType:
		em := not.Value.(*notification.EmailNotification)
		return fmt.Sprintf("%s,%s", nt, em.Email), nil
	case JiraNotificationType:
		jira := not.Value.(*notification.JiraNotification)
		return fmt.Sprintf("%s,%s", nt, jira.CredentialId), nil
	case Office365NotificationType:
		off := not.Value.(*notification.Office365Notification)
		return fmt.Sprintf("%s,%s", nt, off.CredentialId), nil
	case OpsgenieNotificationType:
		og := not.Value.(*notification.OpsgenieNotification)
		return fmt.Sprintf("%s,%s,%s,%s,%s", nt, og.CredentialId, og.ResponderName, og.ResponderId, og.ResponderType), nil
	case PagerDutyNotificationType:
		pd := not.Value.(*notification.PagerDutyNotification)
		return fmt.Sprintf("%s,%s", nt, pd.CredentialId), nil
	case ServiceNowNotificationType:
		sn := not.Value.(*notification.ServiceNowNotification)
		return fmt.Sprintf("%s,%s", nt, sn.CredentialId), nil
	case SlackNotificationType:
		sl := not.Value.(*notification.SlackNotification)
		return fmt.Sprintf("%s,%s,%s", nt, sl.CredentialId, sl.Channel), nil
	case TeamNotificationType:
		t := not.Value.(*notification.TeamNotification)
		return fmt.Sprintf("%s,%s", nt, t.Team), nil
	case TeamEmailNotificationType:
		te := not.Value.(*notification.TeamEmailNotification)
		return fmt.Sprintf("%s,%s", nt, te.Team), nil
	case VictorOpsNotificationType:
		vo := not.Value.(*notification.VictorOpsNotification)
		return fmt.Sprintf("%s,%s,%s", nt, vo.CredentialId, vo.RoutingKey), nil
	case WebhookNotificationType:
		wh := not.Value.(*notification.WebhookNotification)
		return fmt.Sprintf("%s,%s,%s,%s", nt, wh.CredentialId, wh.Secret, wh.Url), nil
	case XMattersNotificationType:
		xm := not.Value.(*notification.XMattersNotification)
		return fmt.Sprintf("%s,%s", nt, xm.CredentialId), nil
	default:
		return "", fmt.Errorf("Unknown notification type %q", nt)
	}
}

func getNotifications(tfNotifications []interface{}) ([]*notification.Notification, error) {
	notificationsList := make([]*notification.Notification, len(tfNotifications))
	for i, tfNotification := range tfNotifications {
		vars := strings.Split(tfNotification.(string), ",")

		var n interface{}

		switch vars[0] {
		case BigPandaNotificationType:
			n = &notification.BigPandaNotification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		case EmailNotificationType:
			n = &notification.EmailNotification{
				Type:  vars[0],
				Email: vars[1],
			}
		case JiraNotificationType:
			n = &notification.JiraNotification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		case Office365NotificationType:
			n = &notification.Office365Notification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		case OpsgenieNotificationType:
			n = &notification.OpsgenieNotification{
				Type:          vars[0],
				CredentialId:  vars[1],
				ResponderName: vars[2],
				ResponderId:   vars[3],
				ResponderType: vars[4],
			}
		case PagerDutyNotificationType:
			n = &notification.PagerDutyNotification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		case ServiceNowNotificationType:
			n = &notification.ServiceNowNotification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		case SlackNotificationType:
			n = &notification.SlackNotification{
				Type:         vars[0],
				CredentialId: vars[1],
				Channel:      vars[2],
			}
		case TeamNotificationType:
			n = &notification.TeamNotification{
				Type: vars[0],
				Team: vars[1],
			}
		case TeamEmailNotificationType:
			n = &notification.TeamEmailNotification{
				Type: vars[0],
				Team: vars[1],
			}
		case VictorOpsNotificationType:
			n = &notification.VictorOpsNotification{
				Type:         vars[0],
				CredentialId: vars[1],
				RoutingKey:   vars[2],
			}
		case WebhookNotificationType:
			n = &notification.WebhookNotification{
				Type:         vars[0],
				CredentialId: vars[1],
				Secret:       vars[2],
				Url:          vars[3],
			}
		case XMattersNotificationType:
			n = &notification.XMattersNotification{
				Type:         vars[0],
				CredentialId: vars[1],
			}
		default:
			return nil, fmt.Errorf("Unknown notification type %q", vars[0])
		}

		item := &notification.Notification{
			Type:  vars[0],
			Value: n,
		}

		notificationsList[i] = item
	}

	return notificationsList, nil
}

func validateNotification(val interface{}, key string) (warns []string, errs []error) {
	parts := strings.Split(val.(string), ",")
	partCount := len(parts)
	if partCount < 2 {
		errs = append(errs, fmt.Errorf("Invalid notification string, not enough commas"))
		return
	}

	switch parts[0] {
	case BigPandaNotificationType, JiraNotificationType, Office365NotificationType, ServiceNowNotificationType, PagerDutyNotificationType, TeamNotificationType, TeamEmailNotificationType, XMattersNotificationType:
		// These are ok, but have no further validation
	case EmailNotificationType:
		if !strings.Contains(parts[1], "@") {
			errs = append(errs, fmt.Errorf("No @ detected in %q, bad email?", parts[1]))
			return
		}
	case OpsgenieNotificationType:
		if partCount != 5 {
			errs = append(errs, fmt.Errorf("Invalid OpsGenie notification string, please consult the documentation (not enough parts)"))
			return
		}
	case SlackNotificationType:
		if partCount != 3 {
			errs = append(errs, fmt.Errorf("Invalid Slack notification string, please consult the documentation (not enough parts)"))
			return
		}
		if strings.Contains(parts[2], "#") {
			errs = append(errs, fmt.Errorf("Please exclude the # from channel names in %q", parts[2]))
			return
		}
	case VictorOpsNotificationType:
		if partCount != 3 {
			errs = append(errs, fmt.Errorf("Invalid VictorOps notification string, please consult the documentation (not enough parts)"))
			return
		}
	case WebhookNotificationType:
		if partCount != 4 {
			errs = append(errs, fmt.Errorf("Invalid Webhook notification string, please consult the documentation (not enough parts)"))
			return
		}
		if parts[1] != "" {
			// We got a credential ID, so verify we didn't get the other parts
			if parts[2] != "" || parts[3] != "" {
				errs = append(errs, fmt.Errorf("Invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)"))
				return
			}
		} else {
			// We didn't get a credential ID so verify we got the other parts
			if parts[2] == "" || parts[3] == "" {
				errs = append(errs, fmt.Errorf("Invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)"))
				return
			}
		}
		// The URL might be an empty string in the case that the user
		// only supplied a credential ID
		if parts[3] != "" {
			_, err := url.ParseRequestURI(parts[3])
			if err != nil {
				errs = append(errs, fmt.Errorf("Invalid Webhook URL %q", parts[2]))
				return
			}
		}
	default:
		errs = append(errs, fmt.Errorf("Invalid notification type %q", parts[0]))
	}

	return
}
