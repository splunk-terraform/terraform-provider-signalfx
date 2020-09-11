package signalfx

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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

    time_range = "-30m"

	grid {
		chart_ids = ["${signalfx_time_chart.mytimechartX0.id}"]
		height = 2
		width = %d
	}
}
`

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
