package signalfx

import (
	"fmt"
	"os"
	"testing"
	"time"

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
			"type":           "Opsgenie",
			"credentialId":   "XXX",
			"credentialName": "YYY",
			"responderName":  "Foo",
			"responderId":    "ABC123",
			"responderType":  "Team",
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
	}

	expected := []string{
		"Email,foo@example.com",
		"Opsgenie,XXX,YYY,Foo,ABC123,Team",
		"PagerDuty,XXX",
		"Slack,XXX,#foobar",
		"Team,ABC123",
		"TeamEmail,ABC123",
		"Webhook,XXX,YYY,http://www.example.com",
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
		"Opsgenie,credId,credName,respName,respId,respType",
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
			"type":           "Opsgenie",
			"credentialId":   "credId",
			"credentialName": "credName",
			"responderName":  "respName",
			"responderId":    "respId",
			"responderType":  "respType",
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
    max_delay = 30

		show_data_markers = true
		show_event_lines = true
		disable_sampling = true
		time_range = 3600
		tags = [ "a", "b" ]

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
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "30"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay",
						"time_range", "3600"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay').max()\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_data_markers", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_event_lines", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.#", "2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.0", "a"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.1", "b"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.notifications.0", "Email,foo-alerts@example.com"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.runbook_url", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1250591008.tip", ""),
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
	// Add some time to let the API quiesce. This may be removed in the future.
	time.Sleep(time.Duration(1) * time.Second)

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
