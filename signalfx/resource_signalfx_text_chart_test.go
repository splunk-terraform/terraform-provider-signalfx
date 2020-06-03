package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newTextChartConfig = `
resource "signalfx_text_chart" "mychartTX" {
  name = "Fart Text"
  description = "Farts"
  markdown = "**farts**"
}
`

const updatedTextChartConfig = `
resource "signalfx_text_chart" "mychartTX" {
  name = "Fart Text NEW"
  description = "Farts NEW"
  markdown = "**farts**"
}
`

func TestAccCreateUpdateTextChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTextChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newTextChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTextChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "name", "Fart Text"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "description", "Farts"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "markdown", "**farts**"),
				),
			},
			{
				ResourceName:      "signalfx_text_chart.mychartTX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_text_chart.mychartTX"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedTextChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTextChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "name", "Fart Text NEW"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "description", "Farts NEW"),
				),
			},
		},
	})
}

func testAccCheckTextChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_text_chart":
			chart, err := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding chart %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccTextChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_text_chart":
			chart, _ := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("Found deleted chart %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
