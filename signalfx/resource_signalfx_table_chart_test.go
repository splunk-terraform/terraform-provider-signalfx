package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newTableChartConfig = `
resource "signalfx_table_chart" "mychartTB" {
  name = "Big Table"
  description = "TableTime"
	program_text = "data('cpu.usage.total').publish(label='CPU Total')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	group_by = ["ClusterName"]

}
`

const updatedTableChartConfig = `
resource "signalfx_table_chart" "mychartTB" {
  name = "Table NEW"
  description = "Tabley Time"
	program_text = "data('cpu.usage.total').publish(label='CPU Total')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	group_by = ["ClusterName"]
}
`

func TestAccCreateUpdateTableChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTableChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newTableChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTableChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "name", "Big Table"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "description", "TableTime"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "program_text", "data('cpu.usage.total').publish(label='CPU Total')"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "hide_timestamp", "true"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "group_by.#", "1"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "group_by.0", "ClusterName"),
				),
			},
			{
				ResourceName:      "signalfx_table_chart.mychartTB",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_table_chart.mychartTB"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedTableChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTableChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "name", "Table NEW"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "description", "Tabley Time"),
				),
			},
		},
	})
}

func testAccCheckTableChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_table_chart":
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

func testAccTableChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_table_chart":
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
