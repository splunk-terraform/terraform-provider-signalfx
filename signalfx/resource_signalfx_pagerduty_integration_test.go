package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationPagerDutyConfig = `
resource "signalfx_pagerduty_integration" "pagerduty_myteamXX" {
    name = "PD - My Team"
    enabled = false
    api_key = "1234567890"
}
`

const updatedIntegrationPagerDutyConfig = `
resource "signalfx_pagerduty_integration" "pagerduty_myteamXX" {
    name = "PD - My Team NEW"
    enabled = false
    api_key = "1234567890"
}
`

func TestAccCreateUpdateIntegrationPagerDuty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationPagerDutyDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationPagerDutyConfig,
				Check:  testAccCheckIntegrationPagerDutyResourceExists,
			},
			{
				ResourceName:            "signalfx_pagerduty_integration.pagerduty_myteamXX",
				ImportState:             true,
				ImportStateIdFunc:       testAccStateIdFunc("signalfx_pagerduty_integration.pagerduty_myteamXX"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			// Update It
			{
				Config: updatedIntegrationPagerDutyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationPagerDutyResourceExists,
					resource.TestCheckResourceAttr("signalfx_pagerduty_integration.pagerduty_myteamXX", "name", "PD - My Team NEW"),
				),
			},
		},
	})
}

func testAccCheckIntegrationPagerDutyResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_pagerduty_integration":
			integration, err := client.GetIntegration(context.TODO(), rs.Primary.ID)
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
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_pagerduty_integration":
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
