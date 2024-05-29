package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const payloadTemplate = `{
    "incidentId": "{{{incidentId}}}"
}
`

const newIntegrationWebhookConfig = `
resource "signalfx_webhook_integration" "webhook_myteamXX" {
    name = "Webhook - My Team"
    enabled = false
    url = "https://www.example.com/v1"
    method = "GET"
    payload_template = <<-EOF
%s EOF

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
    url = "https://www.example.com/v2"
    method = "POST"
    payload_template = <<-EOF
%s EOF

    headers {
      header_key = "foo"
      header_value = "foobar"
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
				Config: fmt.Sprintf(newIntegrationWebhookConfig, payloadTemplate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationWebhookResourceExists,
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "name", "Webhook - My Team"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "url", "https://www.example.com/v1"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "method", "GET"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "payload_template", payloadTemplate),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "headers.0.header_key", "foo"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "headers.0.header_value", "bar"),
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
				Config: fmt.Sprintf(updatedIntegrationWebhookConfig, payloadTemplate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationWebhookResourceExists,
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "name", "Webhook - My Team NEW"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "url", "https://www.example.com/v2"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "method", "POST"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "payload_template", payloadTemplate),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "headers.0.header_key", "foo"),
					resource.TestCheckResourceAttr("signalfx_webhook_integration.webhook_myteamXX", "headers.0.header_value", "foobar"),
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
