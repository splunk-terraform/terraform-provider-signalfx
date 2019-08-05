package notification

// Notification properties for an alert sent via email
type EmailNotification struct {
	// Tells SignalFx which external system it should use to send the notification. For an email notification, this is always \"Email\".
	Type string `json:"type"`
	// The destination address for the notification email. SignalFx doesn't validate this address, so you must ensure it's correct before you use it. SignalFx may not store invalid values, and it may try to  send notification email that doesn't have an address. In either case, the notification won't be delivered.
	Email string `json:"email"`
}
