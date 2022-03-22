package signalfx

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func testAccCreateCheckIntegrationResource(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := newTestClient()

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case resourceName:
				integration, err := client.GetIntegration(context.TODO(), rs.Primary.ID)
				if integration["id"].(string) != rs.Primary.ID || err != nil {
					return fmt.Errorf("error finding integration %s: %s", rs.Primary.ID, err)
				}
			default:
				return fmt.Errorf("unexpected resource of type: %s", rs.Type)
			}
		}

		return nil
	}
}

func testAccCreateCheckDestroyIntegrationResource(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := newTestClient()
		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case resourceName:
				integration, _ := client.GetIntegration(context.TODO(), rs.Primary.ID)
				if _, ok := integration["id"]; ok {
					return fmt.Errorf("found deleted integration %s", rs.Primary.ID)
				}
			default:
				return fmt.Errorf("unexpected resource of type: %s", rs.Type)
			}
		}

		return nil
	}
}
