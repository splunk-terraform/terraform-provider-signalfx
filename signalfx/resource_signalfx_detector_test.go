package signalfx

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"

	sfx "github.com/signalfx/signalfx-go"
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
			"channel":      "#foobar",
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
		"Slack,XXX,#foobar",
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
}

func TestGetNotifications(t *testing.T) {
	values := []interface{}{
		"Email,test@yelp.com",
		"PagerDuty,credId",
		"Webhook,test,https://foo.bar.com?user=test&action=alert",
		"Opsgenie,credId,respName,respId,respType",
		"Slack,credId,#channel",
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
			"channel":      "#channel",
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

func TestResourceRuleHash(t *testing.T) {
	// Tests basic and consistent hashing, keys in the maps are sorted
	values := map[string]interface{}{
		"description":  "Test Rule Name",
		"detect_label": "Test Detect Label",
		"severity":     "Critical",
		"disabled":     "true",
	}

	expected := hashcode.String("Test Rule Name-Critical-Test Detect Label-true-")
	assert.Equal(t, expected, resourceRuleHash(values))

	// Test new params in rules
	values = map[string]interface{}{
		"description":           "Test Rule Name",
		"detect_label":          "Test Detect Label",
		"severity":              "Critical",
		"disabled":              "true",
		"parameterized_subject": "Test subject",
		"parameterized_body":    "Test body",
	}

	expected = hashcode.String("Test Rule Name-Critical-Test Detect Label-true-Test body-Test subject-")
	assert.Equal(t, expected, resourceRuleHash(values))

	values = map[string]interface{}{
		"description":           "Test Rule Name",
		"detect_label":          "Test Detect Label",
		"severity":              "Critical",
		"disabled":              "true",
		"parameterized_subject": "Test subject",
		"parameterized_body":    "Test body",
		"runbook_url":           "https://example.com",
		"tip":                   "test tip",
	}

	expected = hashcode.String("Test Rule Name-Critical-Test Detect Label-true-Test body-Test subject-https://example.com-test tip-")
	assert.Equal(t, expected, resourceRuleHash(values))
}

func TestValidateSeverityAllowed(t *testing.T) {
	_, errors := validateSeverity("Critical", "severity")
	assert.Equal(t, len(errors), 0)
}

func TestValidateSeverityNotAllowed(t *testing.T) {
	_, errors := validateSeverity("foo", "severity")
	assert.Equal(t, len(errors), 1)
}

const newDetectorConfig = `
resource "signalfx_detector" "application_delay" {
    name = "max average delay"
    description = "your application is slow"
    max_delay = 30

    program_text = <<-EOF
        signal = data('app.delay').max()
        detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
        EOF
    rule {
        description = "maximum > 60 for 5m"
        severity = "Warning"
        detect_label = "Processing old messages 5m"
        notifications = ["Email,foo-alerts@example.com"]
    }
    rule {
        description = "maximum > 60 for 30m"
        severity = "Critical"
        detect_label = "Processing old messages 30m"
        notifications = ["Email,foo-alerts@example.com"]
    }
}
`

const updatedDetectorConfig = `
resource "signalfx_detector" "application_delay" {
    name = "max average delay UPDATED"
    description = "your application is slowER"
    max_delay = 60

		show_data_markers = true
		show_event_lines = true
		disable_sampling = true
		time_range = 3600

    program_text = <<-EOF
        signal = data('app.delay2').max()
        detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
        EOF
    rule {
        description = "NEW maximum > 60 for 5m"
        severity = "Warning"
        detect_label = "Processing old messages 5m"
        notifications = ["Email,foo-alerts@example.com"]
				runbook_url = "https://www.example.com"
				tip = "reboot it"
    }
    rule {
        description = "NEW maximum > 60 for 30m"
        severity = "Critical"
        detect_label = "Processing old messages 30m"
        notifications = ["Email,foo-alerts@example.com"]
				runbook_url = "https://www.example.com"
    }
}
`

func TestAccCreateUpdateDetector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDetectorDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDetectorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorResourceExists,
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "name", "max average delay"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "description", "your application is slow"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "30"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay').max()\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_data_markers", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_event_lines", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "disable_sampling", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.#", "2"),
					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.description", "maximum > 60 for 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.detect_label", "Processing old messages 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.runbook_url", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.tip", ""),

					// Rule #2
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.description", "maximum > 60 for 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.detect_label", "Processing old messages 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.runbook_url", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.severity", "Critical"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1714348016.tip", ""),
				),
			},
			// Update It
			{
				Config: updatedDetectorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorResourceExists,
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "name", "max average delay UPDATED"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "description", "your application is slowER"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "60"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay",
						"time_range", "3600"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay2').max()\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_data_markers", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_event_lines", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.#", "2"),
					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.description", "NEW maximum > 60 for 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.runbook_url", "https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1162180415.tip", "reboot it"),
					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.description", "NEW maximum > 60 for 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.detect_label", "Processing old messages 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.runbook_url", "https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.severity", "Critical"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.3455453859.tip", ""),
				),
			},
		},
	})
}

func testAccCheckDetectorResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_detector":
			detector, err := client.GetDetector(rs.Primary.ID)
			if detector.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding detector %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccDetectorDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_detector":
			detector, _ := client.GetDetector(rs.Primary.ID)
			if detector != nil {
				return fmt.Errorf("Found deleted detector %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testTimeRangeStateDataV0() map[string]interface{} {
	return map[string]interface{}{
		"time_range": "-1h",
	}
}

func testTimeRangeStateDataV1() map[string]interface{} {
	return map[string]interface{}{
		"time_range": 3600,
	}
}

func TestTimeRangeStateUpgradeV0(t *testing.T) {
	expected := testTimeRangeStateDataV1()
	actual, err := timeRangeStateUpgradeV0(testTimeRangeStateDataV0(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}
