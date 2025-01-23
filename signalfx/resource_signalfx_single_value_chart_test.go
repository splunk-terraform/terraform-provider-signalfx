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

const newSingleValueChartConfig = `
resource "signalfx_single_value_chart" "mychartSVX" {
  name = "CPU Total Idle - Single Value"
  description = "Farts"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

	color_by = "Scale"
	color_scale {
		gt = 40
		color = "cerise"
	}

	color_scale {
		lte = 40
		color = "vivid_yellow"
	}

	viz_options {
		label = "CPU Idle"
		display_name = "CPU Display"
		color = "azure"
		value_unit = "Bit"
		value_prefix = "foo"
		value_suffix = "bar"
	}

  max_delay = 15
  timezone = "Europe/Paris"
  refresh_interval = 1
  max_precision = 2
  unit_prefix = "Binary"
  secondary_visualization = "Sparkline"
	is_timestamp_hidden = true
	show_spark_line = false
}
`

const updatedSingleValueChartConfig = `
resource "signalfx_single_value_chart" "mychartSVX" {
  name = "CPU Total Idle - Single Value NEW"
  description = "Farts NEW"

  program_text = <<-EOF
  data('cpu.total.idle').publish(label='CPU Idle')
  EOF

	color_by = "Scale"
	color_scale {
		gt = 40
		color = "cerise"
	}

	color_scale {
		lte = 40
		color = "vivid_yellow"
	}

	viz_options {
		label = "CPU Idle"
		display_name = "CPU Display"
		color = "azure"
		value_unit = "Bit"
		value_prefix = "foo"
		value_suffix = "bar"
	}

  max_delay = 15
  timezone = "Europe/Paris"
  refresh_interval = 1
  max_precision = 2
  unit_prefix = "Binary"
  secondary_visualization = "Sparkline"
	is_timestamp_hidden = true
	show_spark_line = false
}
`

const invalidSingleValueChart = `
resource "signalfx_single_value_chart" "mychartSVX"{
  name = ""
  program_text = "A = data('cpu.total.idle').publish(label='CPU Idle')"
}
`

func TestAccValidateSingleValueChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccSingleValueChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidSingleValueChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateUpdateSingleValueChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSingleValueChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newSingleValueChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSingleValueChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "name", "CPU Total Idle - Single Value"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "description", "Farts"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "program_text", "data('cpu.total.idle').publish(label='CPU Idle')\n"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "unit_prefix", "Binary"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_by", "Scale"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "max_delay", "15"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "refresh_interval", "1"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "max_precision", "2"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "secondary_visualization", "Sparkline"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "is_timestamp_hidden", "true"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "show_spark_line", "false"),

					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_scale.#", "2"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_scale.0.color", "cerise"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_scale.0.gt", "40"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_scale.1.color", "vivid_yellow"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "color_scale.1.lte", "40"),
				),
			},
			{
				ResourceName:      "signalfx_single_value_chart.mychartSVX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_single_value_chart.mychartSVX"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedSingleValueChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSingleValueChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "name", "CPU Total Idle - Single Value NEW"),
					resource.TestCheckResourceAttr("signalfx_single_value_chart.mychartSVX", "description", "Farts NEW"),
				),
			},
		},
	})
}

func testAccStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return rs.Primary.Attributes["id"], nil
	}
}

func testAccCheckSingleValueChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_single_value_chart":
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

func testAccSingleValueChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_single_value_chart":
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
