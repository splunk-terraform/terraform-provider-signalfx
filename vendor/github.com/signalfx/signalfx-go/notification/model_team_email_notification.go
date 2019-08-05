package notification

// Properties for a notification sent to the team via email
type TeamEmailNotification struct {
	// Tells SignalFx which system it should use to send the notification. For an TeamEmail notification, this is always \"TeamEmail\".
	Type string `json:"type"`
	// The SignalFx-assigned ID of the team that should receive the notification.
	Team string `json:"team,omitempty"`
}
