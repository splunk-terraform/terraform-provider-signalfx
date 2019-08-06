package signalfx

import (
	"fmt"
	"net/url"
	"strings"
)

func getNotifyStringFromAPI(notification map[string]interface{}) (string, error) {
	nt, ok := notification["type"].(string)
	if !ok {
		return "", fmt.Errorf("Missing type field in notification body")
	}
	switch nt {
	case "Email":
		email, ok := notification["email"].(string)
		if !ok {
			return "", fmt.Errorf("Missing email field from Email body")
		}
		return fmt.Sprintf("%s,%s", nt, email), nil
	case "Opsgenie":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		respName, ok := notification["responderName"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderName field from notification body")
		}
		respId, ok := notification["responderId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderId field from notification body")
		}
		respType, ok := notification["responderType"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderType field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s,%s,%s", nt, cred, respName, respId, respType), nil

	case "PagerDuty", "BigPanda", "Office365", "ServiceNow", "XMatters":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		return fmt.Sprintf("%s,%s", nt, cred), nil
	case "Slack":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		channel, ok := notification["channel"].(string)
		if !ok {
			return "", fmt.Errorf("Missing channel field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s", nt, cred, channel), nil
	case "Team", "TeamEmail":
		team, ok := notification["team"].(string)
		if !ok {
			return "", fmt.Errorf("Missing team field from notification body")
		}
		return fmt.Sprintf("%s,%s", nt, team), nil
	case "VictorOps":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		routing, ok := notification["routingKey"].(string)
		if !ok {
			return "", fmt.Errorf("Missing routing key from notification body")
		}
		return fmt.Sprintf("%s,%s,%s", nt, cred, routing), nil
	case "Webhook":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		secret, ok := notification["secret"].(string)
		if !ok {
			return "", fmt.Errorf("Missing secret field from notification body")
		}
		url, ok := notification["url"].(string)
		if !ok {
			return "", fmt.Errorf("Missing url field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s,%s", nt, cred, secret, url), nil
	}

	return "", nil
}

/*
  Get list of notifications from Resource object (a list of strings), and return a list of notification maps
*/
func getNotifications(tf_notifications []interface{}) []map[string]interface{} {
	notifications_list := make([]map[string]interface{}, len(tf_notifications))
	for i, tf_notification := range tf_notifications {
		vars := strings.Split(tf_notification.(string), ",")

		item := make(map[string]interface{})
		item["type"] = vars[0]

		switch vars[0] {
		case "Email":
			item["email"] = vars[1]
		case "PagerDuty", "BigPanda", "Office365", "ServiceNow", "XMatters":
			item["credentialId"] = vars[1]
		case "Slack":
			item["credentialId"] = vars[1]
			item["channel"] = vars[2]
		case "Webhook":
			item["credentialId"] = vars[1]
			item["secret"] = vars[2]
			item["url"] = vars[3]
		case "Team", "TeamEmail":
			item["team"] = vars[1]
		case "Opsgenie":
			item["credentialId"] = vars[1]
			item["responderName"] = vars[2]
			item["responderId"] = vars[3]
			item["responderType"] = vars[4]
		case "VictorOps":
			item["credentialId"] = vars[1]
			item["routingKey"] = vars[2]
		}

		notifications_list[i] = item
	}

	return notifications_list
}

func validateNotification(val interface{}, key string) (warns []string, errs []error) {
	parts := strings.Split(val.(string), ",")
	if len(parts) < 2 {
		errs = append(errs, fmt.Errorf("Invalid notification string, not enough commas"))
		return
	}

	switch parts[0] {
	case "Email":
		if !strings.Contains(parts[1], "@") {
			errs = append(errs, fmt.Errorf("No @ detected in %q, bad email?", parts[1]))
			return
		}
	case "Slack":
		if len(parts) != 3 {
			errs = append(errs, fmt.Errorf("Invalid Slack notification string, please consult the documentation (not enough parts)"))
			return
		}
		if strings.Contains(parts[2], "#") {
			errs = append(errs, fmt.Errorf("Please exclude the # from channel names in %q", parts[2]))
			return
		}
	case "Webhook":
		if len(parts) != 4 {
			errs = append(errs, fmt.Errorf("Invalid Webhook notification string, please consult the documentation (not enough parts)"))
			return
		}
		_, err := url.ParseRequestURI(parts[3])
		if err != nil {
			errs = append(errs, fmt.Errorf("Invalid Webhook URL %q", parts[2]))
			return
		}
	case "Opsgenie":
		if len(parts) != 5 {
			errs = append(errs, fmt.Errorf("Invalid OpsGenie notification string, please consult the documentation (not enough parts)"))
			return
		}
	case "VictorOps":
		if len(parts) != 3 {
			errs = append(errs, fmt.Errorf("Invalid VictorOps notification string, please consult the documentation (not enough parts)"))
			return
		}
	case "PagerDuty", "BigPanda", "Office365", "ServiceNow", "XMatters", "Team", "TeamEmail":
		// These are ok, but have no further validation
	default:
		errs = append(errs, fmt.Errorf("Invalid notification type %q", parts[0]))
	}

	return
}
