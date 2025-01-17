// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const newDashboardGroupConfig = `
resource "signalfx_dashboard_group" "new_dashboard_group" {
    name = "New Dashboard Group"
}
`

const newDashboardGroupWithDashboardConfig = `
resource "signalfx_dashboard_group" "new_dashboard_group" {
    name = "New Dashboard Group"
}

resource "signalfx_text_chart" "new_chart" {
    name = "chart"
    markdown = "chart"
}

resource "signalfx_dashboard" "new_dashboard" {
    name = "New Dashboard"
    dashboard_group = signalfx_dashboard_group.new_dashboard_group.id
    chart {
        chart_id = signalfx_text_chart.new_chart.id
        width = 6
        row = 0
        column = 0
    }
}
`

const newDashboardGroupsWithDashboardAndDashboardMirrorConfig = `
resource "signalfx_dashboard_group" "new_dashboard_group" {
    name = "New Dashboard Group"
}

resource "signalfx_text_chart" "new_chart" {
    name = "chart"
    markdown = "chart"
}

resource "signalfx_dashboard" "new_dashboard" {
    name = "New Dashboard"
    dashboard_group = signalfx_dashboard_group.new_dashboard_group.id
    chart {
        chart_id = signalfx_text_chart.new_chart.id
        width = 6
        row = 0
        column = 0
    }
}

resource "signalfx_dashboard_group" "new_dashboard_group_2" {
    name = "New Dashboard Group 2"

	dashboard {
        dashboard_id = signalfx_dashboard.new_dashboard.id
	}
}
`

const newDashboardGroupsWithDashboardAndDashboardMirrorWithNameConfig = `
resource "signalfx_dashboard_group" "new_dashboard_group" {
    name = "New Dashboard Group"
}

resource "signalfx_text_chart" "new_chart" {
    name = "chart"
    markdown = "chart"
}

resource "signalfx_dashboard" "new_dashboard" {
    name = "New Dashboard"
    dashboard_group = signalfx_dashboard_group.new_dashboard_group.id
    chart {
        chart_id = signalfx_text_chart.new_chart.id
        width = 6
        row = 0
        column = 0
    }
}

resource "signalfx_dashboard_group" "new_dashboard_group_2" {
    name = "New Dashboard Group 2"

	dashboard {
        dashboard_id = signalfx_dashboard.new_dashboard.id
		name_override = "New Dashboard Name"
	}
}
`

const invalidDashboardGroup = `
resource "signalfx_dashboard_group" "invalid_dashboard_group" {
    name = "Invalid dashboard"
	dashboard {
		dashboard_id = ""
	}
}
`

func TestAccValidateDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidDashboardGroup,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

func TestAccCreateDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: newDashboardGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group", "name", "New Dashboard Group"),
				),
			},
		},
	})
}

func TestAccCreateDashboardGroupWithDashboard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: newDashboardGroupWithDashboardConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group", "dashboard.#", "0"),
				),
			},
		},
	})
}

func TestAccCreateDashboardGroupsWithDashboardAndDashboardMirror(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: newDashboardGroupsWithDashboardAndDashboardMirrorConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group", "dashboard.#", "0"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group_2", "dashboard.#", "1"),
				),
			},
			// Must contain the same number of dashboard configs in the second dashboard group
			{
				Config: newDashboardGroupsWithDashboardAndDashboardMirrorConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group", "dashboard.#", "0"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group_2", "dashboard.#", "1"),
				),
			},
		},
	})
}

func TestAccCreateDashboardGroupsWithDashboardAndDashboardWithNameMirror(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: newDashboardGroupsWithDashboardAndDashboardMirrorConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group_2", "dashboard.#", "1"),
				),
			},
			{
				Config: newDashboardGroupsWithDashboardAndDashboardMirrorWithNameConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group_2", "dashboard.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.new_dashboard_group_2", "dashboard.0.name_override", "New Dashboard Name"),
				),
			},
		},
	})
}
