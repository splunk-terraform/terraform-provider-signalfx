package notification

// Notification properties for an alert sent via xMatters
type XMattersNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For an xMatters notification, this is always  \"XMatters\" (with a capital \"X\").
	Type string `json:"type"`
	// xMatters-supplied credential ID that SignalFx uses to authenticate the notification with the xMatters system. Get this value from your xMatters account settings.
	CredentialId string `json:"credentialId"`
}
