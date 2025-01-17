// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const newHeatmapChartConfig = `
resource "signalfx_heatmap_chart" "mychartHX" {
  name = "Fart Heatmap"
  description = "Farts"
	program_text = "data('cpu.total.idle').publish(label='CPU Idle')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	sort_by = "-foo"
	group_by = ["a", "b"]

	color_range {
		min_value = 1
		max_value = 100
		color = "#ff0000"
	}
}
`

const updatedHeatmapChartConfig = `
resource "signalfx_heatmap_chart" "mychartHX" {
  name = "Fart Heatmap NEW"
  description = "Farts NEW"
	program_text = "data('cpu.total.idle').publish(label='CPU Idle')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	sort_by = "-foo"
	group_by = ["a", "b"]

	color_range {
		min_value = 1
		max_value = 100
		color = "#ff0000"
	}
}
`

const invalidHeatmapChart = `
resource "signalfx_heatmap_chart" "mychartHX"{
  name = ""
  program_text = "A = data('cpu.total.idle').publish(label='CPU Idle')"
}
`

func TestAccValidateHeatmapChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccHeatmapChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidHeatmapChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateUpdateHeatmapChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccHeatmapChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newHeatmapChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHeatmapChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "name", "Fart Heatmap"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "description", "Farts"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "program_text", "data('cpu.total.idle').publish(label='CPU Idle')"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "timezone", "Europe/Paris"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "hide_timestamp", "true"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "sort_by", "-foo"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "color_range.#", "1"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "group_by.#", "2"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "group_by.0", "a"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "group_by.1", "b"),
				),
			},
			{
				ResourceName:      "signalfx_heatmap_chart.mychartHX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_heatmap_chart.mychartHX"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedHeatmapChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHeatmapChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "name", "Fart Heatmap NEW"),
					resource.TestCheckResourceAttr("signalfx_heatmap_chart.mychartHX", "description", "Farts NEW"),
				),
			},
		},
	})
}

func testAccCheckHeatmapChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_heatmap_chart":
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

func testAccHeatmapChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_heatmap_chart":
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

func TestValidateHeatmapChartColors(t *testing.T) {
	_, err := validateHeatmapChartColor("blue", "color")
	assert.Equal(t, 0, len(err))
}

func TestValidateHeatmapChartColorsFail(t *testing.T) {
	_, err := validateHeatmapChartColor("whatever", "color")
	assert.Equal(t, 1, len(err))
}
