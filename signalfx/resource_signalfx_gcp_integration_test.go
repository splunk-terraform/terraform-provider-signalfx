// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const newIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteamXX" {
    name = "GCP - My Team"
    enabled = false
    poll_rate = 600
    services = ["compute"]
    include_list = ["labels"]

    project_service_keys {
		    project_id = "gcp_project_id_1"
		    project_key = "secret_key_project_1"
    }

    project_service_keys {
        project_id = "gcp_project_id_2"
        project_key = "secret_key_project_2"
    }
}
`

const updatedIntegrationGCPConfig = `
resource "signalfx_gcp_integration" "gcp_myteamXX" {
    name = "GCP - My Team NEW"
    enabled = false
    poll_rate = 60
    services = ["compute"]
    include_list = ["labels"]

	auth_method = "WORKLOAD_IDENTITY_FEDERATION"

    project_wif_configs {
		    project_id = "gcp_project_id_1"
		    wif_config = "{\"sample\":\"config1\"}"
    }

    project_wif_configs {
		    project_id = "gcp_project_id_1"
		    wif_config = "{\"sample\":\"config1\"}"
    }

    use_metric_source_project_for_quota = true

    custom_metric_type_domains = ["networking.googleapis.com"]

	import_gcp_metrics = false
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
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.0.project_id", "gcp_project_id_1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.0.project_key", "secret_key_project_1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.1.project_id", "gcp_project_id_2"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_service_keys.1.project_key", "secret_key_project_2"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "include_list.#", "1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "include_list.0", "labels"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "poll_rate", "600"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "use_metric_source_project_for_quota", "false"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "import_gcp_metrics", "true"),
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
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "poll_rate", "60"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "use_metric_source_project_for_quota", "true"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "import_gcp_metrics", "false"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "custom_metric_type_domains.#", "1"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "auth_method", "WORKLOAD_IDENTITY_FEDERATION"),
					resource.TestCheckResourceAttr("signalfx_gcp_integration.gcp_myteamXX", "project_wif_configs.#", "2"),
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
			integration, err := client.GetIntegration(context.TODO(), rs.Primary.ID)
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
			integration, _ := client.GetIntegration(context.TODO(), rs.Primary.ID)
			if _, ok := integration["id"]; ok {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
