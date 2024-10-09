// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"testing"
)

const newLogTimelineConfig = `
resource "signalfx_log_timeline" "mychart1" {
  name = "Chart Name"
  description = "Chart Description"
  program_text = "logs(index=['history','main','o11yhipster','splunklogger','summary']).publish()"

  time_range = 900
  default_connection = "Cosmicbat"
}
`

const updatedLogTimelineConfig = `
resource "signalfx_log_timeline" "mychart1" {
  name = "Chart Name NEW"
  description = "Chart Description NEW"
  program_text = "logs().publish()"

  start_time = 1657647022
  end_time = 1657648042
}
`

func TestAccCreateUpdateLogTimeline(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			fmt.Printf("HERE")
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccLogTimelineDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newLogTimelineConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogTimelineResourceExists,
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "name", "Chart Name"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "description", "Chart Description"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "default_connection", "Cosmicbat"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "time_range", "900"),
				),
			},
			{
				ResourceName:      "signalfx_log_timeline.mychart1",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_log_timeline.mychart1"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedLogTimelineConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogTimelineResourceExists,
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "name", "Chart Name NEW"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "description", "Chart Description NEW"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "start_time", "1657647022"),
					resource.TestCheckResourceAttr("signalfx_log_timeline.mychart1", "end_time", "1657648042"),
				),
			},
		},
	})
}

func testAccCheckLogTimelineResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_log_timeline":
			chart, err := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("error finding chart %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccLogTimelineDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_log_timeline":
			chart, _ := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("found deleted chart %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}
