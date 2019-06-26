package signalfx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newListChartConfig = `
resource "signalfx_list_chart" "mychartLX" {
  name = "CPU Total Idle - List"
  description = "Farts"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

  color_by = "Metric"
  max_delay = 15
  disable_sampling = true
  refresh_interval = 1
  max_precision = 2
  sort_by = "-value"
  unit_prefix = "Binary"
  secondary_visualization = "Linear"

  legend_options_fields = [
    {
      property = "collector"
      enabled = false
    }
  ]
}
`

const updatedListChartConfig = `
resource "signalfx_list_chart" "mychartLX" {
  name = "CPU Total Idle - List NEW"
  description = "Farts NEW"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

  color_by = "Metric"
  max_delay = 15
  disable_sampling = true
  refresh_interval = 1
  max_precision = 2
  sort_by = "-value"
  unit_prefix = "Binary"
  secondary_visualization = "Linear"

  legend_options_fields = [
    {
      property = "collector"
      enabled = false
    }
  ]
}
`

func TestAccCreateListChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccListChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newListChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckListChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "name", "CPU Total Idle - List"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "description", "Farts"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "program_text", "data('cpu.total.idle').publish(label='CPU Idle')\n"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "unit_prefix", "Binary"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_by", "Metric"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "max_delay", "15"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "refresh_interval", "1"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "max_precision", "2"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "secondary_visualization", "Linear"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "sort_by", "-value"),

					// Legend Options
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.#", "1"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.0.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.0.property", "collector"),
				),
			},
			// Update Everything
			{
				Config: updatedListChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckListChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "name", "CPU Total Idle - List NEW"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "description", "Farts NEW"),
				),
			},
		},
	})
}

func testAccCheckListChartResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_list_chart":
			chart, err := client.GetChart(rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding chart %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	// Add some time to let the API quiesce. This may be removed in the future.
	time.Sleep(time.Duration(2) * time.Second)

	return nil
}

func testAccListChartDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_list_chart":
			chart, _ := client.GetChart(rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("Found deleted chart %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
