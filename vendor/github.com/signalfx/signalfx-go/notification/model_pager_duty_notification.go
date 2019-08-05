package notification

// Notification properties for an alert sent via PagerDuty
type PagerDutyNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a PagerDuty notification, this is always \"PagerDuty\".
	Type string `json:"type"`
	// PagerDuty-supplied credential ID that SignalFx uses to authenticate the notification with the PagerDuty system. Get this value from your PagerDuty account settings.
	CredentialId string `json:"credentialId"`
}
