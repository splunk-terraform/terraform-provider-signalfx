package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newIntegrationAWSExternalConfig = `
resource "signalfx_aws_external_integration" "aws_ext_myteamXX" {
    name = "AWSFoo"
    enabled = false
}
`

func TestAccCreateUpdateIntegrationAWSExternal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSExternalDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationAWSConfig,
				Check:  testAccCheckIntegrationAWSExternalResourceExists,
			},
			{
				ResourceName:      "signalfx_aws_external_integration.aws_ext_myteamXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_aws_external_integration.aws_ext_myteamXX"),
				ImportStateVerify: true,
				// The API doesn't return this value, so blow it up
				ImportStateVerifyIgnore: []string{"role_arn"},
			},
		},
	})
}

func testAccCheckIntegrationAWSExternalResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_aws_external_integration":
			integration, err := client.GetAWSCloudWatchIntegration(rs.Primary.ID)
			if integration == nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationAWSExternalDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_aws_external_integration":
			integration, _ := client.GetAWSCloudWatchIntegration(rs.Primary.ID)
			if integration != nil {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
