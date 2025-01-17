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

const newEventFeedChartConfig = `
resource "signalfx_event_feed_chart" "mychartEVX" {
  name = "Fart Event Feed"
  description = "Farts"
	program_text = "A = events(eventType='Fart Testing').publish(label='A')"

  time_range = 900
}
`

const updatedEventFeedChartConfig = `
resource "signalfx_event_feed_chart" "mychartEVX" {
  name = "Fart Event Feed NEW"
  description = "Farts NEW"
	program_text = "A = events(eventType='Fart Testing').publish(label='A')"

  time_range = 900
}
`

const invalidEventFeedChart = `
resource "signalfx_event_feed_chart" "mychartEVX"{
  name = ""
  program_text = "A = events(eventType='Fart Testing').publish(label='A')"
}
`

func TestAccValidateEventFeedChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccEventFeedChartDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidEventFeedChart,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateUpdateEventFeedChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccEventFeedChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newEventFeedChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEventFeedChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "name", "Fart Event Feed"),
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "description", "Farts"),
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "program_text", "A = events(eventType='Fart Testing').publish(label='A')"),
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "time_range", "900"),
				),
			},
			{
				ResourceName:      "signalfx_event_feed_chart.mychartEVX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_event_feed_chart.mychartEVX"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedEventFeedChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEventFeedChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "name", "Fart Event Feed NEW"),
					resource.TestCheckResourceAttr("signalfx_event_feed_chart.mychartEVX", "description", "Farts NEW"),
				),
			},
		},
	})
}

func testAccCheckEventFeedChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_event_feed_chart":
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

func testAccEventFeedChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_event_feed_chart":
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
