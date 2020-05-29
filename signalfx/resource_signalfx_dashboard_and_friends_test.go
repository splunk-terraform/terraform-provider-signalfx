package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newDashConfig = `
resource "signalfx_time_chart" "mytimechart0" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_list_chart" "mylistchart0" {
    name = "CPU Total Idle - List"
    description = "Very cool List Chart"
    program_text = <<-EOF
    data("cpu.total.idle").publish()
    EOF
}

resource "signalfx_single_value_chart" "mysvchart0" {
    name = "CPU Total Idle - Single Value"
    description = "Very cool Single Value Chart"

    program_text = <<-EOF
        data("cpu.total.idle").publish()
        EOF
}

resource "signalfx_heatmap_chart" "myheatmapchart0" {
    name = "CPU Total Idle - Heatmap"
    description = "Very cool Heatmap"

    program_text = <<-EOF
        data("cpu.total.idle").publish()
        EOF

		color_scale {
			gt = 40
			color = "cerise"
		}

		color_scale {
			lte = 40
			color = "vivid_yellow"
		}
}

resource "signalfx_text_chart" "mytextchart0" {
    name = "Important Dashboard Note"
    description = "Lorem ipsum dolor sit amet"
    markdown = <<-EOF
		**Farts
		EOF
}

resource "signalfx_event_feed_chart" "myeventfeedchart0" {
  name = "Fart Event Feed"
  description = "Farts"
	program_text = "A = events(eventType='Fart Testing').publish(label='A')"
}

resource "signalfx_dashboard_group" "mydashboardgroup0" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
		// No teams test cuz there's no teams resource yet!
}

resource "signalfx_dashboard" "mydashboard0" {
    name = "My Dashboard Test 1"
		description = "Cool dashboard"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroup0.id}"

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
		event_overlay {
      line = true
      label = "a event overlabel"
      color = "lilac"
      signal = "overlabel"
      type = "detectorEvents"

      source {
        property = "region"
        values = ["uswest-1"]
        negated = true
      }
    }
		selected_event_overlay {
      signal = "overlabel"
      type = "detectorEvents"

      source {
        property = "region"
        values = ["uswest-1"]
        negated = true
      }
    }

		chart {
        chart_id = "${signalfx_time_chart.mytimechart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_list_chart.mylistchart0.id}"
        width = 12
        height = 1
    }
    chart {
        chart_id = "${signalfx_single_value_chart.mysvchart0.id}"
        width = 12
        height = 1
    }
    chart {
        chart_id = "${signalfx_heatmap_chart.myheatmapchart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_text_chart.mytextchart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_event_feed_chart.myeventfeedchart0.id}"
        width = 12
        height = 1
    }
}

resource "signalfx_data_link" "my_data_link" {
		context_dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    property_name = "pname"
    property_value = "pvalue"

    target_signalfx_dashboard {
      is_default = true
      name = "sfx_dash"
			dashboard_group_id = "${signalfx_dashboard_group.mydashboardgroup0.id}"
			dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    }
}

resource "signalfx_data_link" "my_data_link_dash" {
		context_dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    property_name = "pname2"
    property_value = "pvalue"

    target_external_url {
			is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
`

const updatedDashConfig = `
resource "signalfx_time_chart" "mytimechart0" {
    name = "CPU Total Idle"
    description = "Very cool Time Chart"
    program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF
}

resource "signalfx_list_chart" "mylistchart0" {
    name = "CPU Total Idle - List"
    description = "Very cool List Chart"
    program_text = <<-EOF
    data("cpu.total.idle").publish()
    EOF
}

resource "signalfx_single_value_chart" "mysvchart0" {
    name = "CPU Total Idle - Single Value"
    description = "Very cool Single Value Chart"

    program_text = <<-EOF
        data("cpu.total.idle").publish()
        EOF
}

resource "signalfx_heatmap_chart" "myheatmapchart0" {
    name = "CPU Total Idle - Heatmap"
    description = "Very cool Heatmap"

    program_text = <<-EOF
        data("cpu.total.idle").publish()
        EOF

		color_scale {
			gt = 40
			color = "cerise"
		}

		color_scale {
			lte = 40
			color = "vivid_yellow"
		}
}

resource "signalfx_text_chart" "mytextchart0" {
    name = "Important Dashboard Note"
    description = "Lorem ipsum dolor sit amet"
    markdown = <<-EOF
		**Farts
		EOF
}

resource "signalfx_event_feed_chart" "myeventfeedchart0" {
  name = "Fart Event Feed"
  description = "Farts"
	program_text = "A = events(eventType='Fart Testing').publish(label='A')"
}

resource "signalfx_dashboard_group" "mydashboardgroup0" {
    name = "My team dashboard group NEW"
    description = "Cool dashboard group NEW"
		// No teams test cuz there's no teams resource yet!
}

resource "signalfx_dashboard" "mydashboard0" {
    name = "My Dashboard Test 1 NEW"
		description = "Cool dashboard NEW"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroup0.id}"

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
		event_overlay {
      line = true
      label = "a event overlabel"
      color = "lilac"
      signal = "overlabel"
      type = "detectorEvents"

      source {
        property = "region"
        values = ["uswest-1"]
        negated = true
      }
    }
		selected_event_overlay {
      signal = "overlabel"
      type = "detectorEvents"

      source {
        property = "region"
        values = ["uswest-1"]
        negated = true
      }
    }

		chart {
        chart_id = "${signalfx_time_chart.mytimechart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_list_chart.mylistchart0.id}"
        width = 12
        height = 1
    }
    chart {
        chart_id = "${signalfx_single_value_chart.mysvchart0.id}"
        width = 12
        height = 1
    }
    chart {
        chart_id = "${signalfx_heatmap_chart.myheatmapchart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_text_chart.mytextchart0.id}"
        width = 12
        height = 1
    }
		chart {
        chart_id = "${signalfx_event_feed_chart.myeventfeedchart0.id}"
        width = 12
        height = 1
    }
}

resource "signalfx_data_link" "my_data_link" {
		context_dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    property_name = "pname"
    property_value = "pvalue2"

    target_signalfx_dashboard {
      is_default = true
      name = "sfx_dash"
			dashboard_group_id = "${signalfx_dashboard_group.mydashboardgroup0.id}"
			dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    }
}

resource "signalfx_data_link" "my_data_link_dash" {
		context_dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    property_name = "pname2"
    property_value = "pvalue2"

    target_external_url {
			is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
`

func TestAccCreateUpdateDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDashConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "name", "My Dashboard Test 1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "description", "Cool dashboard"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "charts_resolution", "default"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "time_range", "-30m"),
					// Filters
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.apply_if_exist", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.negated", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.property", "collector"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.values.#", "2"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.values.3211103030", "cpu"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "filter.1325118228.values.3846648755", "Diamond"),
					// Variables
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.property", "region"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.description", "a region"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.alias", "theregion"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.apply_if_exist", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.replace_only", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.restricted_suggestions", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.values.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.values.318300922", "uswest-1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "variable.3642329230.values_suggested.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.color", "lilac"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.label", "a event overlabel"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.line", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.signal", "overlabel"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.source.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.source.0.negated", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.source.0.property", "region"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.source.0.values.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.source.0.values.318300922", "uswest-1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "event_overlay.0.type", "detectorEvents"),

					// Selected Event Overlays
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.signal", "overlabel"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.signal", "overlabel"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.source.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.source.0.negated", "true"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.source.0.property", "region"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.source.0.values.#", "1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.source.0.values.318300922", "uswest-1"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "selected_event_overlay.0.type", "detectorEvents"),

					// Charts
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "chart.#", "6"),
					// We're not testing each chart because they aren't stable, TODO?

					// Dashboard Group
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroup0", "description", "Cool dashboard group"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroup0", "name", "My team dashboard group"),
				),
			},
			{
				ResourceName:      "signalfx_dashboard_group.mydashboardgroup0",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_dashboard_group.mydashboardgroup0"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedDashConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupResourceExists,
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "name", "My Dashboard Test 1 NEW"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "description", "Cool dashboard NEW"),

					// Dashboard Group
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroup0", "description", "Cool dashboard group NEW"),
					resource.TestCheckResourceAttr("signalfx_dashboard_group.mydashboardgroup0", "name", "My team dashboard group NEW"),
				),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("SFX_AUTH_TOKEN"); v == "" {
		t.Fatal("SFX_AUTH_TOKEN must be set for acceptance tests")
	}
}

func testAccCheckDashboardGroupResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart", "signalfx_list_chart", "signalfx_single_value_chart", "signalfx_heatmap_chart", "signalfx_text_chart", "signalfx_event_feed_chart":
			chart, err := client.GetChart(rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding chart %s: %s", rs.Primary.ID, err)
			}
		case "signalfx_dashboard":
			dash, err := client.GetDashboard(rs.Primary.ID)
			if dash.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding dashboard %s: %s", rs.Primary.ID, err)
			}
		case "signalfx_dashboard_group":
			dashgroup, err := client.GetDashboardGroup(rs.Primary.ID)
			if dashgroup.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding dashboard group %s: %s", rs.Primary.ID, err)
			}
		case "signalfx_data_link":
			dl, err := client.GetDataLink(rs.Primary.ID)
			if err != nil || dl.Id != rs.Primary.ID {
				return fmt.Errorf("Error finding data link %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccDashboardGroupDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart", "signalfx_list_chart", "signalfx_single_value_chart", "signalfx_heatmap_chart", "signalfx_text_chart", "signalfx_event_feed_chart":
			chart, _ := client.GetChart(rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("Found deleted chart %s", rs.Primary.ID)
			}
		case "signalfx_dashboard":
			dash, _ := client.GetDashboard(rs.Primary.ID)
			if dash != nil {
				return fmt.Errorf("Found deleted dashboard %s", rs.Primary.ID)
			}
		case "signalfx_dashboard_group":
			dashgroup, _ := client.GetDashboardGroup(rs.Primary.ID)
			if dashgroup != nil {
				return fmt.Errorf("Found deleted dashboard group %s", rs.Primary.ID)
			}
		case "signalfx_data_link":
			dl, _ := client.GetDataLink(rs.Primary.ID)
			if dl != nil {
				return fmt.Errorf("Found deleted data link %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
