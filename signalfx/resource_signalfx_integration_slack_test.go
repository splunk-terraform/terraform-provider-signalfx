package signalfx

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newIntegrationSlackConfig = `
resource "signalfx_slack_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    webhook_url = "http://example.com"
}
`

const updatedIntegrationSlackConfig = `
resource "signalfx_slack_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    webhook_url = "http://example.com"
}
`

// func TestAccCreateIntegrationSlack(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationSlackDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationSlackConfig,
// 				Check:  testAccCheckIntegrationSlackResourceExists,
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationSlackConfig,
// 				Check:  testAccCheckIntegrationSlackResourceExists,
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationSlackResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

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
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
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
