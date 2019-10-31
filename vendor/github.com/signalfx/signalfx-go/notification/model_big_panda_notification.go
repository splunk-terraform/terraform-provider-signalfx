package notification

// Notification properties for an alert sent via BigPanda
type BigPandaNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a BigPanda notification, this is always  \"BigPanda\".
	Type string `json:"type"`
	// The SignalFx ID of the integration profile for BigPanda. Use the Integrations API to get the credential ID for your BigPanda integration.
	CredentialId string `json:"credentialId"`
}
