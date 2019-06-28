package signalfx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
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
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "charts_resolution", "DEFAULT"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "time_range", "-30m"),
					resource.TestCheckResourceAttr("signalfx_dashboard.mydashboard0", "tags.#", "0"),
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
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

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
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	// Add some time to let the API quiesce. This may be removed in the future.
	time.Sleep(time.Duration(2) * time.Second)

	return nil
}

func testAccDashboardGroupDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
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
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
