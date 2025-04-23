// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestResourceRuleHash(t *testing.T) {
	// Tests basic and consistent hashing, keys in the maps are sorted
	values := map[string]any{
		"description":  "Test Rule Name",
		"detect_label": "Test Detect Label",
		"severity":     "Critical",
		"disabled":     "true",
	}

	expected := HashCodeString("Test Rule Name-Critical-Test Detect Label-true-")
	assert.Equal(t, expected, resourceRuleHash(values))

	// Test new params in rules
	values = map[string]any{
		"description":           "Test Rule Name",
		"detect_label":          "Test Detect Label",
		"severity":              "Critical",
		"disabled":              "true",
		"parameterized_subject": "Test subject",
		"parameterized_body":    "Test body",
	}

	expected = HashCodeString("Test Rule Name-Critical-Test Detect Label-true-Test body-Test subject-")
	assert.Equal(t, expected, resourceRuleHash(values))

	values = map[string]any{
		"description":           "Test Rule Name",
		"detect_label":          "Test Detect Label",
		"severity":              "Critical",
		"disabled":              "true",
		"parameterized_subject": "Test subject",
		"parameterized_body":    "Test body",
		"runbook_url":           "https://example.com",
		"tip":                   "test tip",
	}

	expected = HashCodeString("Test Rule Name-Critical-Test Detect Label-true-Test body-Test subject-https://example.com-test tip-")
	assert.Equal(t, expected, resourceRuleHash(values))
}

func TestReminderNotificationInRuleHashing(t *testing.T) {
	values := map[string]any{
		"description":  "Test Rule Name",
		"detect_label": "Test Detect Label",
		"severity":     "Critical",
		"disabled":     "true",
		"reminder_notification": []any{
			map[string]any{
				"interval": 5000,
				"timeout":  10000,
				"type":     "TIMEOUT",
			},
		},
	}

	expected := HashCodeString("Test Rule Name-Critical-Test Detect Label-true-interval-5000-timeout-10000-type-TIMEOUT-")
	assert.Equal(t, expected, resourceRuleHash(values))
}

func TestChangesInReminderNotificationRuleHashing(t *testing.T) {
	values := map[string]any{
		"description":  "Test Rule Name",
		"detect_label": "Test Detect Label",
		"severity":     "Critical",
		"disabled":     "true",
		"reminder_notification": []any{
			map[string]any{
				"interval": 5000,
				"timeout":  10000,
				"type":     "TIMEOUT",
			},
		},
	}
	hashWithReminder := resourceRuleHash(values)

	// modify interval
	values["reminder_notification"].([]any)[0].(map[string]any)["interval"] = 7000
	hashWithChangedInterval := resourceRuleHash(values)
	assert.NotEqual(t, hashWithReminder, hashWithChangedInterval)

	// modify timeout
	values["reminder_notification"].([]any)[0].(map[string]any)["timeout"] = 15000
	hashWithChangedTimeout := resourceRuleHash(values)
	assert.NotEqual(t, hashWithChangedInterval, hashWithChangedTimeout)

	// remove reminder_notification altogether
	delete(values, "reminder_notification")
	hashWithoutReminder := resourceRuleHash(values)
	assert.NotEqual(t, hashWithChangedTimeout, hashWithoutReminder)
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
resource "signalfx_team" "detectorTeam" {
    name = "Splunk Team"
    description = "Detector Team"

    notifications_critical = [ "Email,test@example.com" ]
    notifications_default = [ "Webhook,,secret,https://www.example.com" ]
    notifications_info = [ "Webhook,,secret,https://www.example.com/2" ]
    notifications_major = [ "Webhook,,secret,https://www.example.com/3" ]
    notifications_minor = [ "Webhook,,secret,https://www.example.com/4" ]
    notifications_warning = [ "Webhook,,secret,https://www.example.com/5" ]
}

resource "signalfx_detector" "application_delay" {
    name = "max average delay"
    description = "your application is slow"
    max_delay = 30
    min_delay = 15
    tags = ["tag-1","tag-2"]
    teams = [signalfx_team.detectorTeam.id]
    timezone = "Europe/Paris"
    detector_origin = "Standard"

    program_text = <<-EOF
        signal = data('app.delay').max().publish('app delay')
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

		viz_options {
			label = "app delay"
			color = "orange"
			value_unit = "Second"
		}
}
`

const updatedDetectorConfig = `
resource "signalfx_detector" "application_delay" {
    name = "max average delay UPDATED"
    description = "your application is slowER"
    max_delay = 60
    min_delay = 30
    timezone = "Europe/Paris"

    show_data_markers = true
    show_event_lines = true
    disable_sampling = true
    time_range = 3600
	tags = ["tag-1","tag-2","tag-3"]

    program_text = <<-EOF
        signal = data('app.delay2').max().publish('app delay')
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

		viz_options {
			label = "app delay"
			color = "orange"
			value_unit = "Second"
		}
}
`

const secondUpdatedDetectorConfig = `
resource "signalfx_detector" "application_delay" {
    name = "max average delay UPDATED"
    description = "your application is slowER"
    max_delay = 60
    min_delay = 30
    timezone = "Europe/Paris"

    show_data_markers = true
    show_event_lines = true
    disable_sampling = true
    time_range = 3600

    program_text = <<-EOF
        signal = data('app.delay2').max().publish('app delay')
        detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
        EOF
    rule {
        description = "NEW maximum > 60 for 5m"
        severity = "Warning"
        detect_label = "Processing old messages 5m"
        notifications = ["Email,foo-alerts@example.com"]
				runbook_url = "https://www.example.com"
				tip = "reboot it"
    }
		viz_options {
			label = "app delay"
			color = "orange"
			value_unit = "Second"
		}
}
`

const invalidProgramTextConfig = `
resource "signalfx_detector" "high_cpu_utilization" {
    name = "CPU utilization is high"
    description = "The process is taking too much CPU power"

    program_text = <<-EOF
	A = dat('cpu.utilization').mean(by=['sf_metric', 'sfx_realm']).publish(label='A');
	detect(when(A > threshold(10), lasting='2m'), auto_resolve_after='3d').publish('CPU utilization is high')
	EOF

    rule {
        description = "Maximum > 10 for 2m"
        severity = "Warning"
        detect_label = "CPU utilization is high"
        notifications = ["Email,foo-alerts@example.com"]
    }
}
`

const invalidRulesConfig = `
resource "signalfx_detector" "high_cpu_utilization" {
    name = "CPU utilization is high"
    description = "The process is taking too much CPU power"

    program_text = <<-EOF
	A = data('cpu.utilization').mean(by=['sf_metric', 'sfx_realm']).publish(label='A');
	detect(when(A > threshold(10), lasting='2m'), auto_resolve_after='3d').publish('CPU utilization is high')
	EOF

    rule {
        description = "Maximum > 10 for 2minutes"
        severity = "Warning"
        detect_label = "CPU utilization is low"
        notifications = ["Email,foo-alerts@example.com"]
    }
}
`

// invalid AutoDetect customization detector - missing parent_detector_id
const invalidAutoDetectCustomizationConfig = `
resource "signalfx_detector" "high_cpu_utilization" {
    name = "detector from TF"
    max_delay = 30
    min_delay = 15
    tags = ["tag-1","tag-2"]
    timezone = "Europe/Paris"
    detector_origin = "AutoDetectCustomization"

    program_text = <<-EOF
        signal = data('app.delay').max().publish('app delay')
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
			// Check invalid programTextConfig
			{
				Config:      invalidProgramTextConfig,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Unexpected status code"),
			},
			// Check invalid rulesConfig
			{
				Config:      invalidRulesConfig,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Unexpected status code"),
			},
			// Check invalid AutoDetect customization
			{
				Config:      invalidAutoDetectCustomizationConfig,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Unexpected status code"),
			},
			// Validate plan
			{
				Config:             newDetectorConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create It
			{
				Config: newDetectorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorResourceExists,
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "name", "max average delay"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "description", "your application is slow"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.#", "2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.0", "tag-1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.1", "tag-2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "teams.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "30"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "min_delay", "15"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.%", "2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.Processing old messages 30m", "1000"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.Processing old messages 5m", "1000"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.#", "2"),

					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.description", "maximum > 60 for 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.detect_label", "Processing old messages 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.runbook_url", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.tip", ""),

					// Rule #2
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.description", "maximum > 60 for 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.detect_label", "Processing old messages 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.runbook_url", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.severity", "Critical"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.tip", ""),

					// Force sleep before refresh at the end of test execution
					waitBeforeTestStepPlanRefresh,
				),
			},
			{
				ResourceName:      "signalfx_detector.application_delay",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_detector.application_delay"),
				ImportStateVerify: true,
			},
			// Update It
			{
				Config: updatedDetectorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorResourceExists,
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "name", "max average delay UPDATED"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "description", "your application is slowER"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.#", "3"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.0", "tag-1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.1", "tag-2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.2", "tag-3"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "teams.#", "0"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "60"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "min_delay", "30"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay",
						"time_range", "3600"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay2').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_data_markers", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_event_lines", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "disable_sampling", "true"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.%", "2"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.Processing old messages 30m", "1000"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "label_resolutions.Processing old messages 5m", "1000"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.#", "2"),

					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.description", "NEW maximum > 60 for 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.runbook_url", "https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.1.tip", "reboot it"),

					// Rule #2
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.description", "NEW maximum > 60 for 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.detect_label", "Processing old messages 30m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.notifications.#", "1"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.parameterized_body", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.parameterized_subject", ""),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.runbook_url", "https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.severity", "Critical"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.tip", ""),

					// Force sleep before refresh at the end of test execution
					waitBeforeTestStepPlanRefresh,
				),
			},
			// Subsequent Update
			{
				Config: secondUpdatedDetectorConfig,
				Check: resource.ComposeTestCheckFunc(
					waitBeforeTestStepPlanRefresh,
					testAccCheckDetectorResourceExists,
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "name", "max average delay UPDATED"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "description", "your application is slowER"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "tags.#", "0"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "teams.#", "0"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "max_delay", "60"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "min_delay", "30"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay",
						"time_range", "3600"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "program_text", "signal = data('app.delay2').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\n"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_data_markers", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "show_event_lines", "true"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "disable_sampling", "true"),

					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.#", "1"),

					// Rule #1
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.description", "NEW maximum > 60 for 5m"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.severity", "Warning"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.runbook_url", "https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_detector.application_delay", "rule.0.tip", "reboot it"),
				),
			},
		},
	})
}

func waitBeforeTestStepPlanRefresh(s *terraform.State) error {
	// Gives time to the API to properly update info before read them again
	// required to make the acceptance tests always passing, see:
	// https://github.com/splunk-terraform/terraform-provider-signalfx/pull/306#issuecomment-870417521
	time.Sleep(30 * time.Second)
	return nil
}

func testAccCheckDetectorResourceExists(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_detector":
			detector, err := client.GetDetector(context.TODO(), rs.Primary.ID)
			if err != nil || detector.Id != rs.Primary.ID {
				return fmt.Errorf("Error finding detector %s: %s", rs.Primary.ID, err)
			}
		case "signalfx_team":
			team, err := client.GetTeam(context.TODO(), rs.Primary.ID)
			if err != nil || team.Id != rs.Primary.ID {
				return fmt.Errorf("Error finding team %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccDetectorDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_detector":
			detector, _ := client.GetDetector(context.TODO(), rs.Primary.ID)
			if detector != nil {
				return fmt.Errorf("Found deleted detector %s", rs.Primary.ID)
			}
		case "signalfx_team":
			team, _ := client.GetTeam(context.TODO(), rs.Primary.ID)
			if team != nil {
				return fmt.Errorf("Found deleted team %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testTimeRangeStateDataV0() map[string]any {
	return map[string]any{
		"time_range": "-1h",
	}
}

func testTimeRangeStateDataV1() map[string]any {
	return map[string]any{
		"time_range": 3600,
	}
}

func TestTimeRangeStateUpgradeV0(t *testing.T) {
	expected := testTimeRangeStateDataV1()
	actual, err := timeRangeStateUpgradeV0(context.Background(), testTimeRangeStateDataV0(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func TestSerializeReminderToString(t *testing.T) {
	t.Run("Serialize reminder with all fields", func(t *testing.T) {
		reminder := map[string]any{
			"interval": 5000,
			"timeout":  10000,
			"type":     "TIMEOUT",
		}
		expected := "interval-5000-timeout-10000-type-TIMEOUT-"
		assert.Equal(t, expected, serializeReminderToString(reminder))
	})

	t.Run("Serialize reminder with missing optional fields", func(t *testing.T) {
		reminder := map[string]any{
			"interval": 5000,
			"type":     "TIMEOUT",
		}
		expected := "interval-5000-type-TIMEOUT-"
		assert.Equal(t, expected, serializeReminderToString(reminder))
	})

	t.Run("Serialize empty reminder map", func(t *testing.T) {
		reminder := map[string]any{}
		expected := ""
		assert.Equal(t, expected, serializeReminderToString(reminder))
	})

	t.Run("Serialize reminder with unordered keys", func(t *testing.T) {
		reminder := map[string]any{
			"type":     "TIMEOUT",
			"timeout":  10000,
			"interval": 5000,
		}
		expected := "interval-5000-timeout-10000-type-TIMEOUT-"
		assert.Equal(t, expected, serializeReminderToString(reminder))
	})

	t.Run("Serialize reminder with non-string values", func(t *testing.T) {
		reminder := map[string]any{
			"interval": 5000,
			"timeout":  nil,
			"type":     true,
		}
		expected := "interval-5000-timeout-<nil>-type-true-"
		assert.Equal(t, expected, serializeReminderToString(reminder))
	})
}
