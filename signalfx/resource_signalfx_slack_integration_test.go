package signalfx

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationSlackConfig = `
resource "signalfx_slack_integration" "slack_myteamXX" {
    name = "Slack - My Team"
    enabled = false
    webhook_url = "https://example.com"
}
`

const updatedIntegrationSlackConfig = `
resource "signalfx_slack_integration" "slack_myteamXX" {
    name = "Slack - My Team NEW"
    enabled = false
    webhook_url = "https://example.com"
}
`

func TestAccCreateUpdateIntegrationSlack(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationSlackDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationSlackConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationSlackResourceExists,
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "name", "Slack - My Team"),
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "webhook_url", "https://example.com"),
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "enabled", "false"),
				),
			},
			{
				ResourceName:      "signalfx_slack_integration.slack_myteamXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_slack_integration.slack_myteamXX"),
				ImportStateVerify: true,
				// The API doesn't return this value, so blow it up
				ImportStateVerifyIgnore: []string{"webhook_url"},
			},
			// Update It
			{
				Config: updatedIntegrationSlackConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationSlackResourceExists,
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "name", "Slack - My Team NEW"),
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "webhook_url", "https://example.com"),
					resource.TestCheckResourceAttr("signalfx_slack_integration.slack_myteamXX", "enabled", "false"),
				),
			},
		},
	})
}

func testAccCheckIntegrationSlackResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_slack_integration":
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

func testAccIntegrationSlackDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_slack_integration":
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
