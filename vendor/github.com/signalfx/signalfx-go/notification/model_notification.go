package notification

import "encoding/json"

// Notification properties for an alert sent to the team
type Notification struct {
	Type  string
	Value interface{}
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	var typ struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typ); err != nil {
		return err
	}
	n.Type = typ.Type
	switch typ.Type {
	case "BigPanda":
		n.Value = &BigPandaNotification{}
	case "Email":
		n.Value = &EmailNotification{}
	case "Office365":
		n.Value = &Office365Notification{}
	case "OpsGenie":
		n.Value = &OpsgenieNotification{}
	case "PagerDuty":
		n.Value = &PagerDutyNotification{}
	case "ServiceNow":
		n.Value = &ServiceNowNotification{}
	case "Slack":
		n.Value = &SlackNotification{}
	case "Team":
		n.Value = &TeamNotification{}
	case "TeamEmail":
		n.Value = &TeamEmailNotification{}
	case "VictorOps":
		n.Value = &VictorOpsNotification{}
	case "Webhook":
		n.Value = &WebhookNotification{}
	case "XMatters":
		n.Value = &XMattersNotification{}
	}
	return json.Unmarshal(data, n.Value)
}

// MarshalJSON unwraps the `Value` and makes this marshal in the way we expect.
func (n *Notification) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.Value)
}
