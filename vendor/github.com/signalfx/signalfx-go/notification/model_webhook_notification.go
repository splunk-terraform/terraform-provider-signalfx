package notification

// Notification properties for an alert sent via a Webhook
type WebhookNotification struct {
	// Tells SignalFx which system to use to send the notification. For a Webhook notification, this is always \"Webhook\".
	Type string `json:"type"`
	// Authentication credentials for the system that the Webhook url  connects to. You usually find this value in the account settings for the system that the Webhook connects to.
	CredentialId string `json:"credentialId,omitempty"`
	// A secret value that identifies the Webhook integration to use when  sending notifications. This value also indicates that the  notification has permission to use the integration.  If `credentialId` is set, this property is ignored.
	Secret string `json:"secret,omitempty"`
	// The URL of a Webhook integration. You must provide the mechanism for processing notifications sent to the URL and routing them to the proper chat or incident management system. If `credentialId` is set, this property is ignored.
	Url string `json:"url,omitempty"`
}
