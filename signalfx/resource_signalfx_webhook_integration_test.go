package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationWebhookConfig = `
resource "signalfx_webhook_integration" "webhook_myteamXX" {
    name = "Webhook - My Team"
    enabled = true
    url = "https://www.example.com"
    shared_secret = "poot"

    headers {
      header_key = "foo"
      header_value = "bar"
    }
}
`

const updatedIntegrationWebhookConfig = `
resource "signalfx_webhook_integration" "webhook_myteamXX" {
    name = "Webhook - My Team NEW"
    enabled = true
    url = "https://www.example.com"
    shared_secret = "poot"

    headers {
      header_key = "foo"
      header_value = "bar"
    }
}
`

func TestAccCreateUpdateIntegrationWebhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationWebhookDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationWebhookConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationWebhookResourceExists,
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "name", "Webhook - My Team"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "enabled", "true"),
				),
			},
			{
				ResourceName:      "signalfx_webhook_integration.webhook_myteamXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_webhook_integration.webhook_myteamXX"),
				ImportStateVerify: true,
			},
			// Update It
			{
				Config: updatedIntegrationWebhookConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationWebhookResourceExists,
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "name", "Webhook - My Team NEW"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckIntegrationWebhookResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_webhook_integration":
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

func testAccIntegrationWebhookDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_webhook_integration":
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
