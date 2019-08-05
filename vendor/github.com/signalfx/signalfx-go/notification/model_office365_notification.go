package notification

// Notification properties for an alert sent via Office365
type Office365Notification struct {
	// Tells SignalFx which external system it should use to send the notification. For an Office365 notification, this is always  \"Office365\".
	Type string `json:"type"`
	// The SignalFx ID of the integration profile for Office365. Use the Integrations API to get the credential ID for your Office365 integration.
	CredentialId string `json:"credentialId"`
}
