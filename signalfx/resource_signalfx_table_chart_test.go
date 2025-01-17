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

const newTableChartConfig = `
resource "signalfx_table_chart" "mychartTB" {
  name = "Big Table"
  description = "TableTime"
	program_text = "data('cpu.usage.total').publish(label='CPU Total')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	group_by = ["ClusterName"]

	viz_options {
		label = "CPU Total"
		display_name = "CPU Total Display"
		value_unit = "Bit"
		value_prefix = "foo"
		value_suffix = "bar"
	}
}
`

const updatedTableChartConfig = `
resource "signalfx_table_chart" "mychartTB" {
  name = "Table NEW"
  description = "Tabley Time"
	program_text = "data('cpu.usage.total').publish(label='Updated CPU Total')"

	disable_sampling = true
	timezone = "Europe/Paris"
	hide_timestamp = true
	group_by = ["ClusterName"]

	viz_options {
		label = "Updated CPU Total"
		display_name = "Updated CPU Total Display"
		value_unit = "Bit"
		value_prefix = "Updated foo"
		value_suffix = "Updated bar"
	}
}
`

const invalidTableChartChart = `
resource "signalfx_table_chart" "mychartTB"{
  name = ""
  program_text = "A = data('cpu.usage.total').publish(label='Updated CPU Total')"
}
`

func TestAccValidateTableChartChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccTableChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidTableChartChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

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
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.#", "1"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.label", "CPU Total"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.display_name", "CPU Total Display"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_unit", "Bit"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_prefix", "foo"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_suffix", "bar"),
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
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.#", "1"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.label", "Updated CPU Total"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.display_name", "Updated CPU Total Display"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_prefix", "Updated foo"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_suffix", "Updated bar"),
					resource.TestCheckResourceAttr("signalfx_table_chart.mychartTB", "viz_options.0.value_suffix", "Updated bar"),
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
