package notification

// Notification properties for an alert sent via Jira
type JiraNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a Jira notification, this is always  \"Jira\".
	Type string `json:"type"`
	// The SignalFx ID of the integration profile for Jira. Use the Integrations API to get the credential ID for your Jira integration.
	CredentialId string `json:"credentialId"`
}
