package signalfx

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newIntegrationPagerDutyConfig = `
resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
    name = "PD - My Team"
    enabled = true
    api_key = "1234567890"
}
`

const updatedIntegrationPagerDutyConfig = `
resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
    name = "PD - My Team 2"
    enabled = true
    api_key = "1234567890"
}
`

// Commented out because SignalFx validates incoming keys and ours aren't valid.
// func TestAccCreateIntegrationPagerDuty(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationPagerDutyDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationPagerDutyConfig,
// 				Check:  testAccCheckIntegrationPagerDutyResourceExists,
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationPagerDutyConfig,
// 				Check:  testAccCheckIntegrationPagerDutyResourceExists,
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationPagerDutyResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_pagerduty_integration":
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

func testAccIntegrationPagerDutyDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_pagerduty_integration":
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
