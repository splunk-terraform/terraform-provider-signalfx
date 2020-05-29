package signalfx

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/stretchr/testify/assert"
)

const newIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteamXX" {
    name = "GCP - My Team"
    enabled = false
    poll_rate = 300
    services = ["compute"]
    whitelist = ["labels"]

    project_service_keys {
		    project_id = "gcp_project_id_1"
		    project_key = "secret_farts"
    }

    project_service_keys {
        project_id = "gcp_project_id_2"
        project_key = "secret_farts_2"
    }
}
`

const updatedIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteamXX" {
    name = "GCP - My Team NEW"
    enabled = false
    poll_rate = 300
    services = ["compute"]
    whitelist = ["labels"]

    project_service_keys {
		    project_id = "gcp_project_id_1"
		    project_key = "secret_farts"
    }

    project_service_keys {
        project_id = "gcp_project_id_2"
        project_key = "secret_farts_2"
    }
}
`

func TestAccCreateUpdateIntegrationGCP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationGCPDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationGCPConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationGCPResourceExists,
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.#", "2"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.1428645045.project_id", "gcp_project_id_1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.1428645045.project_key", "secret_farts"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.605621665.project_id", "gcp_project_id_2"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.605621665.project_key", "secret_farts_2"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "whitelist.#", "1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "whitelist.151844697", "labels"),
				),
			},
			{
				ResourceName:      "signalfx_gcp_integration.gcp_myteamXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_gcp_integration.gcp_myteamXX"),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"project_service_keys",
				},
			},
			// Update It
			{
				Config: updatedIntegrationGCPConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationGCPResourceExists,
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "name", "GCP - My Team NEW"),
				),
			},
		},
	})
}

func testAccCheckIntegrationGCPResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_gcp_integration":
			integration, err := client.GetIntegration(rs.Primary.ID)
			id := integration["id"]
			if id != nil && id.(string) != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationGCPDestroy(s *terraform.State) error {
	client := newTestClient()
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

func TestValidateGcpService(t *testing.T) {
	_, errors := validateGcpService("appengine", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateGcpService("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}
