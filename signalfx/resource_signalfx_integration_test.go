package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"

	sfx "github.com/signalfx/signalfx-go"
)

func TestValidatePollRateAllowed(t *testing.T) {
	for _, value := range []int{60000, 300000} {
		_, errors := validatePollRate(value, "poll_rate")
		assert.Equal(t, 0, len(errors))
	}
}

func TestValidatePollRateNotAllowed(t *testing.T) {
	_, errors := validatePollRate(1000, "poll_rate")
	assert.Equal(t, 1, len(errors))
}

const newIntegrationConfig = `
resource "signalfx_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    type = "Slack"
    webhook_url = "http://example.com"
}
`

const updatedIntegrationConfig = `
resource "signalfx_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    type = "Slack"
    webhook_url = "http://example.com"
}
`

// func TestAccCreateIntegration(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationConfig,
// 				Check:  testAccCheckIntegrationResourceExists,
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationConfig,
// 				Check:  testAccCheckIntegrationResourceExists,
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_integration":
			integration, err := client.GetIntegration(rs.Primary.ID)
			if integration["id"].(string) != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_integration":
			integration, _ := client.GetIntegration(rs.Primary.ID)
			if _, ok := integration["id"]; ok {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
