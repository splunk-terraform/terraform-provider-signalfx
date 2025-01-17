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

const newTextChartConfig = `
resource "signalfx_text_chart" "mychartTX" {
  name = "Chart Name"
  description = "Chart Description"
  markdown = "**chart markdown**"
}
`

const updatedTextChartConfig = `
resource "signalfx_text_chart" "mychartTX" {
  name = "Chart Name NEW"
  description = "Chart Description NEW"
  markdown = "**chart markdown**"
}
`

const invalidTextChartChart = `
resource "signalfx_text_chart" "mychartTX"{
  name = ""
  markdown = "**chart markdown**"
}
`

func TestAccValidateTextChartChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccTextChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidTextChartChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

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
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "name", "Chart Name"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "description", "Chart Description"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "markdown", "**chart markdown**"),
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
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "name", "Chart Name NEW"),
					resource.TestCheckResourceAttr("signalfx_text_chart.mychartTX", "description", "Chart Description NEW"),
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
