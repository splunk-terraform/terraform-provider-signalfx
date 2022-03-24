package signalfx

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

const newDashConfigConfig = `
resource "signalfx_time_chart" "mytimechartX0" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_team" "dashboardGroupTeam" {
    name = "Super Cool Team"
    description = "Dashboard Group Team"

    notifications_critical = [ "Email,test@example.com" ]
    notifications_default = [ "Webhook,,secret,https://www.example.com" ]
    notifications_info = [ "Webhook,,secret,https://www.example.com/2" ]
    notifications_major = [ "Webhook,,secret,https://www.example.com/3" ]
    notifications_minor = [ "Webhook,,secret,https://www.example.com/4" ]
    notifications_warning = [ "Webhook,,secret,https://www.example.com/5" ]
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

		filter {
        property = "collector"
        values = ["cpu", "Diamond"]
        negated = true
        apply_if_exist = true
    }
		variable {
	      property = "region"
	      description = "a region"
	      alias = "theregion"
	      apply_if_exist = true
	      values = ["uswest-1"]
	      value_required = true
	      values_suggested = ["uswest-1"]
	      restricted_suggestions = true
	      replace_only = true
    }

		chart {
        chart_id = "${signalfx_time_chart.mytimechartX0.id}"
        width = 12
        height = 1
    }
}

resource "signalfx_dashboard_group" "mydashboardgroupX1" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
    teams = [signalfx_team.dashboardGroupTeam.id]

    // Test Mirrors!
    dashboard {
      dashboard_id = "${signalfx_dashboard.mydashboardX0.id}"
      name_override = "FART"
      description_override = "GAS MASTER"
    }
}
`

const updatedDashConfigConfig = `
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

		filter {
        property = "collector"
        values = ["cpu", "Diamond"]
        negated = true
        apply_if_exist = true
    }
		variable {
	      property = "region"
	      description = "a region"
	      alias = "theregion"
	      apply_if_exist = true
	      values = ["uswest-1"]
	      value_required = true
	      values_suggested = ["uswest-1"]
	      restricted_suggestions = true
	      replace_only = true
    }

		chart {
        chart_id = "${signalfx_time_chart.mytimechartX0.id}"
        width = 12
        height = 1
    }
}

resource "signalfx_dashboard_group" "mydashboardgroupX1" {
    name = "My group with a mirror"
    description = "Mirror having dashboard group"

    // Test Mirrors!
    dashboard {
      dashboard_id = "${signalfx_dashboard.mydashboardX0.id}"
      name_override = "FART NEW"
      description_override = "GAS MASTER NEW"

      filter_override {
        property = "collector"
        values = [ "foo" ]
        negated = true
      }

      variable_override {
        property = "region"
        values = ["foo"]
        values_suggested = ["foo", "bar"]
      }
    }
}
`

func TestAccCreateUpdateDashboardGroupWithConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDashConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.name_override", "FART"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.description_override", "GAS MASTER"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.#", "0"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.#", "0"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "teams.#", "1"),
				),
			},
			// Update Everything
			{
				Config: updatedDashConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.name_override", "FART NEW"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.description_override", "GAS MASTER NEW"),
					// Filters
					resource.TestCheckResourceAttrSet("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.dashboard_id"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.#", "1"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.0.negated", "true"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.0.property", "collector"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.values.#", "1"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.filter_override.values.0", "foo"),
					// Variables
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.#", "1"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.property", "region"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.values.#", "1"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.values.0", "foo"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.values_suggested.#", "2"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.values_suggested.0", "foo"),
					// resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroupX1", "dashboard.0.variable_override.0.values_suggested.1", "bar"),
				),
			},
		},
	})
}
