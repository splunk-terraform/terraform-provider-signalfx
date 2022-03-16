package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/stretchr/testify/assert"
)

const newIntegrationAzureConfig = `
resource "signalfx_azure_integration" "azure_int" {
    name = "AzureFoo"
    enabled = false

    environment = "azure"

    poll_rate = 120

    secret_key = "XXX"

    app_id = "YYY"

    tenant_id = "ZZZ"

    services = [ "microsoft.sql/servers/elasticpools" ]

    subscriptions = [ "microsoft.sql/servers/elasticpools" ]
}
`

const updatedIntegrationAzureConfig = `
resource "signalfx_azure_integration" "azure_int" {
    name = "AzureFoo NEW"
    enabled = false

    environment = "azure"

    poll_rate = 600

    secret_key = "XXX"

    app_id = "YYY"

    tenant_id = "ZZZ"

    services = [ "microsoft.sql/servers/elasticpools" ]

    additional_services = [ "foo", "bar" ]

    subscriptions = [ "microsoft.sql/servers/elasticpools" ]

    resource_filter_rules {
        filter = {
            source = "filter('azure_tag_service', 'payment') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"
        }
    }
    resource_filter_rules {
        filter = {
            source = "filter('azure_tag_service', 'notification') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"
        }
    }
}
`

func TestAccCreateUpdateIntegrationAzure(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAzureDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationAzureConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAzureResourceExists,
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "poll_rate", "120"),
				),
			},
			{
				ResourceName:      "signalfx_azure_integration.azure_int",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_azure_integration.azure_int"),
				ImportStateVerify: true,
				// The API doesn't return this value, so blow it up
				ImportStateVerifyIgnore: []string{"app_id", "secret_key"},
			},
			// Update It
			{
				Config: updatedIntegrationAzureConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAzureResourceExists,
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "name", "AzureFoo NEW"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "poll_rate", "600"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "additional_services.#", "2"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "additional_services.0", "foo"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "additional_services.1", "bar"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "resource_filter_rules.#", "2"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "resource_filter_rules.0.filter.source",
						"filter('azure_tag_service', 'payment') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"),
					resource.TestCheckResourceAttr("signalfx_azure_integration.azure_int", "resource_filter_rules.1.filter.source",
						"filter('azure_tag_service', 'notification') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"),
				),
			},
		},
	})
}

func testAccCheckIntegrationAzureResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_azure_integration":
			integration, err := client.GetAzureIntegration(context.TODO(), rs.Primary.ID)
			if integration == nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationAzureDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_azure_integration":
			integration, _ := client.GetAzureIntegration(context.TODO(), rs.Primary.ID)
			if integration != nil {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func TestValidateAzureService(t *testing.T) {
	_, errors := validateAzureService("microsoft.batch/batchaccounts", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAzureService("unknown/service", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}
