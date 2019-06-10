package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"

	sfx "github.com/signalfx/signalfx-go"
)

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
        signal = data('app.delay', filter('cluster','prod'), extrapolation='last_value', maxExtrapolations=5).max()
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
    name = "max average delay"
    description = "your application is slow"
    max_delay = 30
    program_text = <<-EOF
        signal = data('app.delay', filter('cluster','prod'), extrapolation='last_value', maxExtrapolations=5).max()
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

func TestAccCreateDetector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDetectorDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDetectorConfig,
				Check:  testAccCheckDetectorResourceExists,
			},
			// Update It
			{
				Config: updatedDetectorConfig,
				Check:  testAccCheckDetectorResourceExists,
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
			if detector.Id != "" {
				return fmt.Errorf("Found deleted detector %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
