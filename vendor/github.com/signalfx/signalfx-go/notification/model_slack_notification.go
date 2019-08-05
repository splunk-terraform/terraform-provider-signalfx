package notification

// Notification properties for an alert sent via Slack
type SlackNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For a Slack notification, this is always \"Slack\".
	Type string `json:"type"`
	// The name of the Slack channel in which to display the notification. Omit the leading \"#\" symbol. For example, specify \"#critical-notifications\" as \"critical-notifications\".
	Channel string `json:"channel"`
	// Slack-supplied credential ID that SignalFx uses to authenticate the notification with the Slack system. Get this value from your Slack account settings.
	CredentialId string `json:"credentialId"`
}
