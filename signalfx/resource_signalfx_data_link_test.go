package signalfx

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newDataLinkConfig = `
resource "signalfx_data_link" "big_test_data_link" {
    property_name = "pname"
    property_value = "pvalue"

    target_external_url {
      is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
`

const newDataLinkConfigWithoutPropertyValue = `
resource "signalfx_data_link" "big_test_data_link" {
    property_name = "pname"

    target_external_url {
      is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
`

const newDataLinkConfigWithoutPropertyErr = `
resource "signalfx_data_link" "big_test_data_link" {
    target_signalfx_dashboard {
      dashboard_group_id = "XYZ"
      dashboard_id = "XYZ"
      is_default = false
      name = "dashboard"
    }
}
`

const updatedDataLinkConfig = `
resource "signalfx_data_link" "big_test_data_link" {
    property_name = "pname"
    property_value = "pvalue_new"

    target_external_url {
      is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
`

func TestAccCreateDashboardDataLinkFails(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDataLinkDestroy,
		Steps: []resource.TestStep{
			{
				Config:      newDataLinkConfigWithoutPropertyErr,
				ExpectError: regexp.MustCompile("Must supply a property_name when using target_signalfx_dashboard"),
			},
		},
	})

}

func TestAccCreateUpdateDataLink(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataLinkDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDataLinkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataLinkResourceExists,
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_name", "pname"),
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_value", "pvalue"),
				),
			},
			{
				ResourceName:      "signalfx_data_link.big_test_data_link",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_data_link.big_test_data_link"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedDataLinkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataLinkResourceExists,
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_name", "pname"),
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_value", "pvalue_new"),
				),
			},
		},
	})
}

func TestAccCreateUpdateDataLinkWithoutPropertyValue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataLinkDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDataLinkConfigWithoutPropertyValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataLinkResourceExists,
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_name", "pname"),
					resource.TestCheckResourceAttr("signalfx_data_link.big_test_data_link", "property_value", ""),
				),
			},
		},
	})
}

func testAccCheckDataLinkResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_data_link":
			fmt.Printf("[DEBUG] SignalFx: GETTING DATA LINK %s", rs.Primary.ID)
			dl, err := client.GetDataLink(rs.Primary.ID)
			if err != nil || dl.Id != rs.Primary.ID {
				return fmt.Errorf("Error finding data link %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccDataLinkDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_data_link":
			dl, _ := client.GetDataLink(rs.Primary.ID)
			if dl != nil {
				return fmt.Errorf("Found deleted data link %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
