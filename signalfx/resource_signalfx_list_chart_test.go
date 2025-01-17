// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const newListChartConfig = `
resource "signalfx_list_chart" "mychartLX" {
  name = "CPU Total Idle - List"
  description = "Farts"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

  max_delay = 15
  timezone = "Europe/Paris"
  disable_sampling = true
  hide_missing_values = true
  refresh_interval = 1
  max_precision = 2
  sort_by = "-value"
  unit_prefix = "Binary"
  secondary_visualization = "Sparkline"

	color_by = "Scale"
	color_scale {
		gt = 40
		color = "cerise"
	}

	color_scale {
		lte = 40
		color = "vivid_yellow"
	}

	legend_options_fields {
		property = "collector"
		enabled  = false
	}

	viz_options {
		label = "CPU Idle"
		display_name = "CPU Idle Display"
		color = "azure"
		value_unit = "Bit"
		value_prefix = "foo"
		value_suffix = "bar"
	}
}
`

const updatedListChartConfig = `
resource "signalfx_list_chart" "mychartLX" {
  name = "CPU Total Idle - List NEW"
  description = "Farts NEW"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

  max_delay = 15
  timezone = "Europe/Paris"
  disable_sampling = true
  hide_missing_values = true
  refresh_interval = 1
  max_precision = 2
  sort_by = "-value"
  unit_prefix = "Binary"
	secondary_visualization = "Sparkline"

	color_by = "Scale"
	color_scale {
		gt = 40
		color = "cerise"
	}

	color_scale {
		lte = 40
		color = "vivid_yellow"
	}

	legend_options_fields {
		property = "collector"
		enabled  = false
	}

	viz_options {
		label = "CPU Idle"
		display_name = "CPU Idle Display"
		color = "azure"
		value_unit = "Bit"
		value_prefix = "foo"
		value_suffix = "bar"
	}
}
`

const invalidListChart = `
resource "signalfx_list_chart" "mychartLX"{
  name = ""
  program_text = "A = data('demo.lb.hosts').publish(label='A')"
}
`

func TestAccValidateListChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccListChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidListChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateUpdateListChart(t *testing.T) {
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
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_by", "Scale"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "max_delay", "15"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "hide_missing_values", "true"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "refresh_interval", "1"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "max_precision", "2"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "secondary_visualization", "Sparkline"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "sort_by", "-value"),

					// Legend Options
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.#", "1"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.0.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "legend_options_fields.0.property", "collector"),

					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_scale.#", "2"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_scale.0.color", "cerise"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_scale.0.gt", "40"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_scale.1.color", "vivid_yellow"),
					resource.TestCheckResourceAttr("signalfx_list_chart.mychartLX", "color_scale.1.lte", "40"),
				),
			},
			{
				ResourceName:      "signalfx_list_chart.mychartLX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_list_chart.mychartLX"),
				ImportStateVerify: true,
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
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_list_chart":
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

func testAccListChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_list_chart":
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
