package notification

// Notification properties for an alert sent via VictorOps
type VictorOpsNotification struct {
	// Tells SignalFx which system to use to send the notification. For a VictorOps notification, this is always \"VictorOps\".
	Type string `json:"type"`
	// VictorOps-supplied credential ID that SignalFx uses to authenticate the notification with the VictorOps system. Get this value from your VictorOps account settings.
	CredentialId string `json:"credentialId"`
	// Indicates the routing key used to determine how to process the  notification message. This key specifies where the notification is posted and how related alerts are escalated. For more information see the  [VictorOps knowlegebase](https://help.victorops.com/knowledge-base/routing-keys/).
	RoutingKey string `json:"routingKey"`
}
