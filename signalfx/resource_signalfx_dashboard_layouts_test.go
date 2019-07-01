package signalfx

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

const gridDashLayoutConfig = `
resource "signalfx_time_chart" "mytimechartLAYOUT1" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_dashboard_group" "mydashboardgroupLAYOUT1" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
		// No teams test cuz there's no teams resource yet!
}

resource "signalfx_dashboard" "mydashboardLAYOUT1" {
    name = "My Dashboard Test 1"
		description = "Cool dashboard"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroupLAYOUT1.id}"

    grid {
        chart_ids = [ "${signalfx_time_chart.mytimechartLAYOUT1.id}" ]
        width = 3
        height = 1
    }
}
`

const columnDashLayoutConfig = `
resource "signalfx_time_chart" "mytimechartLAYOUT2" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_dashboard_group" "mydashboardgroupLAYOUT2" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
		// No teams test cuz there's no teams resource yet!
}

resource "signalfx_dashboard" "mydashboardLAYOUT2" {
    name = "My Dashboard Test 1"
		description = "Cool dashboard"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroupLAYOUT2.id}"

    column {
        chart_ids = [ "${signalfx_time_chart.mytimechartLAYOUT2.id}" ]
        width = 2
        column = 4
    }
}
`

func TestAccCreateUpdateDashboardGridLayout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: gridDashLayoutConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT1", "name", "My Dashboard Test 1"),

					// Charts
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT1", "grid.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT1", "grid.0.chart_ids.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT1", "grid.0.height", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT1", "grid.0.width", "3"),
				),
			},
		},
	})
}

func TestAccCreateUpdateDashboardColumnLayout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: columnDashLayoutConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "name", "My Dashboard Test 1"),

					// Charts
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "column.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "column.0.chart_ids.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "column.0.column", "4"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "column.0.height", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboardLAYOUT2", "column.0.width", "2"),
				),
			},
		},
	})
}
