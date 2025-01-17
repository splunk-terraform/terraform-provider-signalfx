// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
)

const widthTestDashConfig = `
resource "signalfx_time_chart" "mytimechartX0" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_dashboard_group" "mydashboardgroupX0" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
}

resource "signalfx_dashboard" "mydashboardX0" {
    name = "My Dashboard Test 1"
    description = "Cool dashboard"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroupX0.id}"
    tags = ["cool tag", "not so cool tag"]

    time_range = "-30m"

	grid {
		chart_ids = ["${signalfx_time_chart.mytimechartX0.id}"]
		height = 2
		width = %d
	}
}
`

const invalidDashboard = `
resource "signalfx_dashboard" "invalid_dashboard" {
  name = ""
  dashboard_group = ""
}
`

func TestAccValidateDashboard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidDashboard,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestValidateChartsResolutionAllowed(t *testing.T) {
	for _, value := range []string{"default", "low", "high", "highest"} {
		_, errors := validateChartsResolution(value, "charts_resolution")
		assert.Equal(t, len(errors), 0)
	}
}

func TestValidateChartsResolutionNotAllowed(t *testing.T) {
	_, errors := validateChartsResolution("whatever", "charts_resolution")
	assert.Equal(t, len(errors), 1)
}

func TestChartWidthAllowed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create resource with minimum value
			{
				Config: fmt.Sprintf(widthTestDashConfig, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "grid.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "grid.0.width", "1"),
				),
			},
			// Update resource with maximum value
			{
				Config: fmt.Sprintf(widthTestDashConfig, 12),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "grid.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "grid.0.width", "12"),
				),
			},
		},
	})
}

func TestDashboardTagsApplied(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create resource with tags
			{
				Config: fmt.Sprintf(widthTestDashConfig, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "tags.#", "2"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "tags.0", "cool tag"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardX0", "tags.1", "not so cool tag"),
				),
			},
		},
	})
}
