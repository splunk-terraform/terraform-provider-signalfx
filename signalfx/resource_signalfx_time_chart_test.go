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

const newTimeChartConfig = `
resource "signalfx_time_chart" "mychartXX" {
    name = "CPU Total Idle"
		description = "I am described"

    program_text = <<-EOF
data('cpu.total.idle').publish(label='CPU Idle')
events(eventType='some.testing').publish(label='testing events')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true
		timezone = "Europe/Paris"

    plot_type = "Histogram"
		show_event_lines = true
		stacked = false
		axes_precision = 4

		on_chart_legend_dimension = "plot_label"

		legend_options_fields {
			property = "collector"
			enabled  = false
		}
    viz_options {
        label = "CPU Idle"
				display_name = "CPU Idle Display"
        axis = "left"
        color = "orange"
				plot_type = "Histogram"
				value_unit = "Byte"
				value_prefix = "prefix"
				value_suffix = "suffix"
    }
		event_options {
			label = "testing events"
			display_name = "events display name"
			color = "azure"
		}

		histogram_options {
			color_theme = "lilac"
		}

    axis_left {
        label = "OMG on fire"
				high_watermark = 2000
				high_watermark_label = "high"
        low_watermark = 1000
				low_watermark_label = "low"
				min_value = 900
				max_value = 2100
    }

		axis_right {
        label = "OMG still on fire"
				high_watermark = 2001
				high_watermark_label = "higher"
        low_watermark = 1001
				low_watermark_label = "lower"
				min_value = 901
				max_value = 2101
    }
}

resource "signalfx_time_chart" "mychartXY" {
    name = "CPU Total Idle"
		description = "I am described"

    program_text = <<-EOF
        data('cpu.total.idle').publish(label='CPU Idle')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true
		timezone = "Europe/Paris"

    plot_type = "LineChart"
    show_data_markers = true
		show_event_lines = true
		stacked = false
		axes_precision = 4

		legend_options_fields {
			property = "collector"
			enabled  = false
		}
}
`

const updatedTimeChartConfig = `
resource "signalfx_time_chart" "mychartXX" {
    name = "CPU Total Idle NEW"
		description = "I am described NEW"

    program_text = <<-EOF
data('cpu.total.idle').publish(label='CPU Idle')
events(eventType='some.testing').publish(label='testing events')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true
		timezone = "Europe/Paris"

    plot_type = "LineChart"
		show_event_lines = true
		stacked = false
		axes_precision = 4

		legend_options_fields {
			property = "collector"
			enabled  = false
		}
    viz_options {
        label = "CPU Idle"
				display_name = "CPU Idle Display"
        axis = "left"
        color = "orange"
				plot_type = "Histogram"
				value_unit = "Byte"
				value_prefix = "prefix"
				value_suffix = "suffix"
    }
		event_options {
			label = "testing events"
			display_name = "events display name"
			color = "azure"
		}

		histogram_options {
			color_theme = "lilac"
		}

    axis_left {
        label = "OMG on fire"
				high_watermark = 2000
				high_watermark_label = "high"
        low_watermark = 1000
				low_watermark_label = "low"
				min_value = 900
				max_value = 2100
    }

		axis_right {
        label = "OMG still on fire"
				high_watermark = 2001
				high_watermark_label = "higher"
        low_watermark = 1001
				low_watermark_label = "lower"
				min_value = 901
				max_value = 2101
    }
}

resource "signalfx_time_chart" "mychartXY" {
    name = "CPU Total Idle NEW"
		description = "I am described NEW"

    program_text = <<-EOF
        data('cpu.total.idle').publish(label='CPU Idle')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true
		timezone = "Europe/Paris"

    plot_type = "LineChart"
    show_data_markers = true
		show_event_lines = true
		stacked = false
		axes_precision = 4

		legend_options_fields {
			property = "collector"
			enabled  = false
		}
}
`

const invalidTimeChartChart = `
resource "signalfx_time_chart" "mychartXY"{
  name = ""
  program_text = "A = data('cpu.total.idle').publish(label='CPU Idle')"
}
`

func TestAccValidateTimeChartChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccTimeChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidTimeChartChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateUpdateTimeChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTimeChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newTimeChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTimeChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "name", "CPU Total Idle"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "description", "I am described"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXY", "name", "CPU Total Idle"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXY", "description", "I am described"),
				),
			},
			{
				ResourceName:      "signalfx_time_chart.mychartXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_time_chart.mychartXX"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedTimeChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTimeChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "name", "CPU Total Idle NEW"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "description", "I am described NEW"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXY", "name", "CPU Total Idle NEW"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXY", "description", "I am described NEW"),
				),
			},
		},
	})
}

func testAccCheckTimeChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart":
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

func testAccTimeChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart":
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
