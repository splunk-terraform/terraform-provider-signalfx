package signalfx

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationOpsgenieConfig = `
resource "signalfx_opsgenie_integration" "opsgenie_myteamXX" {
    name = "Opsgenie - My Team"
    enabled = false
    api_key = "fart"
    api_url = "https://api.opsgenie.com"
}
`

const updatedIntegrationOpsgenieConfig = `
resource "signalfx_opsgenie_integration" "opsgenie_myteamXX" {
    name = "Opsgenie - My Team NEW"
    enabled = false
    api_key = "fart"
    api_url = "https://api.opsgenie.com"
}
`

// Commented out because SignalFx seems to validate this integration even if
// it is disabled.
// func TestAccCreateUpdateIntegrationOpsgenie(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationOpsgenieDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationOpsgenieConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckIntegrationOpsgenieResourceExists,
// 					resource.TestCheckResourceAttr("signalfx_opsgenie_integration.opsgenie_myteamXX", "name", "Opsgenie - My Team"),
// 					resource.TestCheckResourceAttr("signalfx_opsgenie_integration.opsgenie_myteamXX", "enabled", "false"),
// 				),
// 			},
// 			{
// 				ResourceName:      "signalfx_opsgenie_integration.opsgenie_myteamXX",
// 				ImportState:       true,
// 				ImportStateIdFunc: testAccStateIdFunc("signalfx_opsgenie_integration.opsgenie_myteamXX"),
// 				ImportStateVerify: true,
// 				// The API doesn't return this value, so blow it up
// 				ImportStateVerifyIgnore: []string{"api_url", "api_key"},
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationOpsgenieConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckIntegrationOpsgenieResourceExists,
// 					resource.TestCheckResourceAttr("signalfx_opsgenie_integration.opsgenie_myteamXX", "name", "Opsgenie - My Team NEW"),
// 					resource.TestCheckResourceAttr("signalfx_opsgenie_integration.opsgenie_myteamXX", "enabled", "false"),
// 				),
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationOpsgenieResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_opsgenie_integration":
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

func testAccIntegrationOpsgenieDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_opsgenie_integration":
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
