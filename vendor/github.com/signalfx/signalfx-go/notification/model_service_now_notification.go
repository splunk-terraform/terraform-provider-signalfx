package notification

// Notification properties for an alert sent via ServiceNow
type ServiceNowNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a ServiceNow notification, this is always \"ServiceNow\".
	Type string `json:"type"`
	// ServiceNow-supplied credential ID that SignalFx uses to authenticate the notification with the ServiceNow system. Get this value from your ServiceNow account settings.
	CredentialId string `json:"credentialId"`
}
