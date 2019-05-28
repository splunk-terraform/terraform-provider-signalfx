package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

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
