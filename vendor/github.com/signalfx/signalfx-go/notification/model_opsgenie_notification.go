package notification

// Notification properties for an alert sent via Opsgenie.<br> **NOTE:**<br> When you specify Opsgenie as the notification service type, use  either the `responderId` or `responderName` request body property, but don't use both.
type OpsgenieNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For an Opsgenie notification, this is always \"Opsgenie\".
	Type string `json:"type"`
	// Opsgenie-supplied credential ID that SignalFx uses to authenticate the notification with the Opsgenie system. Get this value from your Opsgenie account settings.
	CredentialId string `json:"credentialId"`
	// Name of an Opsgenie entity to which SignalFx assigns the alert<br> **NOTE:**<br> Specify either `responderName` or `responderId`, but not both.
	ResponderName string `json:"responderName,omitempty"`
	// ID of an Opsgenie entity to which SignalFx assigns the alert<br> **NOTE:**<br> Specify either `responderId` or `responderName`, but not both.
	ResponderId string `json:"responderId,omitempty"`
	// Type of the Opsgenie entity specified by `responderName`. The only valid value is `\"Team\"`.
	ResponderType string `json:"responderType,omitempty"`
}
