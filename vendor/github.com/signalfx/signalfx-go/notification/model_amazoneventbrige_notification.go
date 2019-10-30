package notification

// Notification properties for an alert sent via AWS EventBrige
type AmazonEventBrigeNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a Amazon EventBrige notification, this is always \"AmazonEventBrige\".
	Type string `json:"type"`
	// The SignalFx ID of the integration profile for Amazon EventBrige. Use the Integrations API to get the credential ID for your AmazonEventBrige integration.
	CredentialId string `json:"credentialId"`
}
