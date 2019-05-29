package signalfx

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteam" {
    name = "GCP - My Team"
    enabled = true
    poll_rate = 300000
    services = ["compute"]
    project_service_keys = [
        {
            project_id = "gcp_project_id_1"
            project_key = "secret_farts"
        },
        {
            project_id = "gcp_project_id_2"
            project_key = "secret_farts_2"
        }
    ]
}
`

const updatedIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteam" {
    name = "GCP - My Team 2"
    enabled = true
    poll_rate = 300000
    services = ["compute"]
    project_service_keys = [
        {
            project_id = "gcp_project_id_1"
            project_key = "secret_farts"
        },
        {
            project_id = "gcp_project_id_2"
            project_key = "secret_farts_2"
        }
    ]
}
`

// Commented out because SignalFx validates incoming keys and ours aren't valid.
// func TestAccCreateIntegrationGCP(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testAccIntegrationGCPDestroy,
// 		Steps: []resource.TestStep{
// 			// Create It
// 			{
// 				Config: newIntegrationGCPConfig,
// 				Check:  testAccCheckIntegrationGCPResourceExists,
// 			},
// 			// Update It
// 			{
// 				Config: updatedIntegrationGCPConfig,
// 				Check:  testAccCheckIntegrationGCPResourceExists,
// 			},
// 		},
// 	})
// }

func testAccCheckIntegrationGCPResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_gcp_integration":
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

func testAccIntegrationGCPDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_gcp_integration":
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
