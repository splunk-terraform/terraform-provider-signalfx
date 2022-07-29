package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newLogsListChartConfig = `
resource "signalfx_logs_list_chart" "mychart0" {
  name = "Chart Name"
  description = "Chart Description"
  program_text = <<-EOF
  logs(index=['history','main','o11yhipster','splunklogger','summary']).publish()
  EOF

  time_range = 900
  default_connection = "Cosmicbat"
    columns  {
      name= "severity"
    }
    columns  {
        name= "time"
    }
    columns  {
      name= "_raw"
    }
  sort_options {
    descending= false
    field= "severity"
   }
}
`

const updatedLogsListChartConfig = `
resource "signalfx_logs_list_chart" "mychart0" {
  name = "Chart Name NEW"
  description = "Chart Description NEW"
  program_text = <<-EOF
  logs().publish()
  EOF

  start_time = 1657647022
  end_time = 1657648042

    columns  {
      name= "severity"
    }
    columns  {
        name= "time"
    }
    columns  {
      name= "_raw"
    }
  sort_options {
    descending= true
    field= "severity"
   }
}
`

func TestAccCreateUpdateLogsListChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccLogsListChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newLogsListChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogsListChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "name", "Chart Name"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "description", "Chart Description"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "columns", "**chart markdown**"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "sort_options", "**chart markdown**"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "default_connection", "Cosmicbat"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "time_range", "**chart markdown**"),
				),
			},
			{
				ResourceName:      "signalfx_logs_list_chart.mychart0",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_logs_list_chart.mychart0"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedLogsListChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogsListChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "name", "Chart Name NEW"),
					resource.TestCheckResourceAttr("signalfx_logs_list_chart.mychart0", "description", "Chart Description NEW"),
				),
			},
		},
	})
}

func testAccCheckLogsListChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_logs_list_chart":
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

func testAccLogsListChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_logs_list_chart":
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
