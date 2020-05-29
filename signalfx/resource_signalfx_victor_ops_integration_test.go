package signalfx

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationVictorOpsConfig = `
resource "signalfx_victor_ops_integration" "victor_ops_myteamXX" {
    name = "VictorOps - My Team"
    enabled = false
    post_url = "https://alert.victorops.com/integrations/generic/123/alert/$key/$routing_key"
}
`

const updatedIntegrationVictorOpsConfig = `
resource "signalfx_victor_ops_integration" "victor_ops_myteamXX" {
    name = "VictorOps - My Team NEW"
    enabled = false
    post_url = "https://alert.victorops.com/integrations/generic/123/alert/$key/$routing_key"
}
`

// Commented out because SignalFx seems to validate this integration even if
// it is disabled.
// func TestAccCreateUpdateIntegrationVictorOps(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationVictorOpsDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationVictorOpsConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckIntegrationVictorOpsResourceExists,
// 					resource.TestCheckResourceAttr("signalfx_victor_ops_integration.victor_ops_myteamXX", "name", "VictorOps - My Team"),
// 					resource.TestCheckResourceAttr("signalfx_victor_ops_integration.victor_ops_myteamXX", "enabled", "false"),
// 				),
// 			},
// 			{
// 				ResourceName:      "signalfx_victor_ops_integration.victor_ops_myteamXX",
// 				ImportState:       true,
// 				ImportStateIdFunc: testAccStateIdFunc("signalfx_victor_ops_integration.victor_ops_myteamXX"),
// 				ImportStateVerify: true,
// 				// The API doesn't return this value, so blow it up
// 				ImportStateVerifyIgnore: []string{"post_url"},
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationVictorOpsConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckIntegrationVictorOpsResourceExists,
// 					resource.TestCheckResourceAttr("signalfx_victor_ops_integration.victor_ops_myteamXX", "name", "VictorOps - My Team NEW"),
// 					resource.TestCheckResourceAttr("signalfx_victor_ops_integration.victor_ops_myteamXX", "enabled", "false"),
// 				),
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationVictorOpsResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_victor_ops_integration":
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

func testAccIntegrationVictorOpsDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_victor_ops_integration":
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
