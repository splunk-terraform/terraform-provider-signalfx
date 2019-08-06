package signalfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifyStringFromAPI(t *testing.T) {
	values := []map[string]interface{}{
		{
			"type":  "Email",
			"email": "foo@example.com",
		},
		{
			"type":          "Opsgenie",
			"credentialId":  "XXX",
			"responderName": "Foo",
			"responderId":   "ABC123",
			"responderType": "Team",
		},
		{
			"type":         "PagerDuty",
			"credentialId": "XXX",
		},
		{
			"type":         "Slack",
			"credentialId": "XXX",
			"channel":      "foobar",
		},
		{
			"type": "Team",
			"team": "ABC123",
		},
		{
			"type": "TeamEmail",
			"team": "ABC123",
		},
		{
			"type":         "Webhook",
			"credentialId": "XXX",
			"secret":       "YYY",
			"url":          "http://www.example.com",
		},
		{
			"type":         "BigPanda",
			"credentialId": "XXX",
		},
		{
			"type":         "Office365",
			"credentialId": "XXX",
		},
		{
			"type":         "ServiceNow",
			"credentialId": "XXX",
		},
		{
			"type":         "VictorOps",
			"credentialId": "XXX",
			"routingKey":   "YYY",
		},
		{
			"type":         "XMatters",
			"credentialId": "XXX",
		},
	}

	expected := []string{
		"Email,foo@example.com",
		"Opsgenie,XXX,Foo,ABC123,Team",
		"PagerDuty,XXX",
		"Slack,XXX,foobar",
		"Team,ABC123",
		"TeamEmail,ABC123",
		"Webhook,XXX,YYY,http://www.example.com",
		"BigPanda,XXX",
		"Office365,XXX",
		"ServiceNow,XXX",
		"VictorOps,XXX,YYY",
		"XMatters,XXX",
	}

	for i, v := range values {
		result, err := getNotifyStringFromAPI(v)
		assert.NoError(t, err, "Got error making notify string")
		assert.Equal(t, expected[i], result)
	}

	for _, v := range expected {
		_, errors := validateNotification(v, "notification")
		assert.Len(t, errors, 0, "Expected no errors from valid notification")
	}
}

func TestGetNotifications(t *testing.T) {
	values := []interface{}{
		"Email,test@yelp.com",
		"PagerDuty,credId",
		"Webhook,test,https://foo.bar.com?user=test&action=alert",
		"Opsgenie,credId,respName,respId,respType",
		"Slack,credId,channel",
		"Team,teamId",
		"TeamEmail,teamId",
		"BigPanda,credId",
		"Office365,credId",
		"ServiceNow,credId",
		"VictorOps,credId,routingKey",
		"XMatters,credId",
	}

	expected := []map[string]interface{}{
		map[string]interface{}{
			"type":  "Email",
			"email": "test@yelp.com",
		},
		map[string]interface{}{
			"type":         "PagerDuty",
			"credentialId": "credId",
		},
		map[string]interface{}{
			"type":   "Webhook",
			"secret": "test",
			"url":    "https://foo.bar.com?user=test&action=alert",
		},
		map[string]interface{}{
			"type":          "Opsgenie",
			"credentialId":  "credId",
			"responderName": "respName",
			"responderId":   "respId",
			"responderType": "respType",
		},
		map[string]interface{}{
			"type":         "Slack",
			"credentialId": "credId",
			"channel":      "channel",
		},
		map[string]interface{}{
			"type": "Team",
			"team": "teamId",
		},
		map[string]interface{}{
			"type": "TeamEmail",
			"team": "teamId",
		},
		map[string]interface{}{
			"type":         "BigPanda",
			"credentialId": "credId",
		},
		map[string]interface{}{
			"type":         "Office365",
			"credentialId": "credId",
		},
		map[string]interface{}{
			"type":         "ServiceNow",
			"credentialId": "credId",
		},
		map[string]interface{}{
			"type":         "VictorOps",
			"credentialId": "credId",
			"routingKey":   "routingKey",
		},
		map[string]interface{}{
			"type":         "XMatters",
			"credentialId": "credId",
		},
	}
	assert.Equal(t, expected, getNotifications(values))
}
