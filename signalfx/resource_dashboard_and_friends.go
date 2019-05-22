package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newDashConfig = `
resource "signalfx_time_chart" "mychart0" {
    name = "CPU Total Idle"

    program_text = <<-EOF
        myfilters = filter("shc_name", "prod") and filter("role", "splunk_searchhead")
        data("cpu.total.idle", filter=myfilters).publish(label="CPU Idle")
        EOF

    time_range = "-15m"

    plot_type = "LineChart"
    show_data_markers = true

    legend_fields_to_hide = ["collector", "prefix", "hostname"]
    viz_options {
        label = "CPU Idle"
        axis = "left"
        color = "orange"
    }

    axis_left {
        label = "CPU Total Idle"
        low_watermark = 1000
    }
}

resource "signalfx_list_chart" "mylistchart0" {
    name = "CPU Total Idle - List"

    program_text = <<-EOF
    myfilters = filter("cluster_name", "prod") and filter("role", "search")
    data("cpu.total.idle", filter=myfilters).publish()
    EOF

    description = "Very cool List Chart"

    color_by = "Metric"
    max_delay = 2
    disable_sampling = true
    refresh_interval = 1
    legend_fields_to_hide = ["collector", "host"]
    max_precision = 2
    sort_by = "-value"
 }

 resource "signalfx_single_value_chart" "mysvchart0" {
    name = "CPU Total Idle - Single Value"

    program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

    description = "Very cool Single Value Chart"

    color_by = "Dimension"

    max_delay = 2
    refresh_interval = 1
    max_precision = 2
    is_timestamp_hidden = true
}

resource "signalfx_heatmap_chart" "myheatmapchart0" {
    name = "CPU Total Idle - Heatmap"

    program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

    description = "Very cool Heatmap"

    disable_sampling = true
    sort_by = "+host"
    group_by = ["hostname", "host"]
    hide_timestamp = true
}

resource "signalfx_text_chart" "mynote0" {
    name = "Important Dashboard Note"
    description = "Lorem ipsum dolor sit amet, laudem tibique iracundia at mea. Nam posse dolores ex, nec cu adhuc putent honestatis"

    markdown = <<-EOF
    1. First ordered list item
    2. Another item
      * Unordered sub-list.
    1. Actual numbers don't matter, just that it's a number
      1. Ordered sub-list
    4. And another item.

       You can have properly indented paragraphs within list items. Notice the blank line above, and the leading spaces (at least one, but we'll use three here to also align the raw Markdown).

       To have a line break without a paragraph, you will need to use two trailing spaces.⋅⋅
       Note that this line is separate, but within the same paragraph.⋅⋅
       (This is contrary to the typical GFM line break behaviour, where trailing spaces are not required.)

    * Unordered list can use asterisks
    - Or minuses
    + Or pluses
    EOF
}

resource "signalfx_dashboard_group" "mydashboardgroup0" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
}

resource "signalfx_dashboard" "mydashboard0" {
    name = "My Dashboard Test 1"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroup0.id}"

    time_range = "-30m"

    filter {
        property = "collector"
        values = ["cpu", "Diamond"]
    }
    variable {
        property = "region"
        alias = "region"
        values = ["uswest-1-"]
    }
    chart {
        chart_id = "${signalfx_time_chart.mychart0.id}"
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
        chart_id = "${signalfx_text_chart.mynote0.id}"
        width = 12
        height = 1
    }
}
`

const updatedDashConfig = `
resource "signalfx_time_chart" "mychart0" {
    name = "CPU Total Idle NEW"

    program_text = <<-EOF
        myfilters = filter("shc_name", "prod") and filter("role", "splunk_searchhead")
        data("cpu.total.idle", filter=myfilters).publish(label="CPU Idle")
        EOF

    time_range = "-15m"

    plot_type = "LineChart"
    show_data_markers = true

    legend_fields_to_hide = ["collector", "prefix", "hostname"]
    viz_options {
        label = "CPU Idle"
        axis = "left"
        color = "orange"
    }

    axis_left {
        label = "CPU Total Idle"
        low_watermark = 1000
    }
}

resource "signalfx_list_chart" "mylistchart0" {
    name = "CPU Total Idle - List NEW"

    program_text = <<-EOF
    myfilters = filter("cluster_name", "prod") and filter("role", "search")
    data("cpu.total.idle", filter=myfilters).publish()
    EOF

    description = "Very cool List Chart"

    color_by = "Metric"
    max_delay = 2
    disable_sampling = true
    refresh_interval = 1
    legend_fields_to_hide = ["collector", "host"]
    max_precision = 2
    sort_by = "-value"
}

resource "signalfx_single_value_chart" "mysvchart0" {
    name = "CPU Total Idle - Single Value NEW"

    program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

    description = "Very cool Single Value Chart"

    color_by = "Dimension"

    max_delay = 2
    refresh_interval = 1
    max_precision = 2
    is_timestamp_hidden = true
}

resource "signalfx_heatmap_chart" "myheatmapchart0" {
     name = "CPU Total Idle - Heatmap NEW"

     program_text = <<-EOF
         myfilters = filter("cluster_name", "prod") and filter("role", "search")
         data("cpu.total.idle", filter=myfilters).publish()
         EOF

     description = "Very cool Heatmap"

     disable_sampling = true
     sort_by = "+host"
     group_by = ["hostname", "host"]
     hide_timestamp = true
}

resource "signalfx_text_chart" "mynote0" {
    name = "Important Dashboard Note NEW"
    description = "Lorem ipsum dolor sit amet, laudem tibique iracundia at mea. Nam posse dolores ex, nec cu adhuc putent honestatis"

    markdown = <<-EOF
    1. First ordered list item
    2. Another item
      * Unordered sub-list.
    1. Actual numbers don't matter, just that it's a number
      1. Ordered sub-list
    4. And another item.

       You can have properly indented paragraphs within list items. Notice the blank line above, and the leading spaces (at least one, but we'll use three here to also align the raw Markdown).

       To have a line break without a paragraph, you will need to use two trailing spaces.⋅⋅
       Note that this line is separate, but within the same paragraph.⋅⋅
       (This is contrary to the typical GFM line break behaviour, where trailing spaces are not required.)

    * Unordered list can use asterisks
    - Or minuses
    + Or pluses
    EOF
}

resource "signalfx_dashboard_group" "mydashboardgroup0" {
    name = "My team dashboard group NEW"
    description = "Cool dashboard group "
}

resource "signalfx_dashboard" "mydashboard0" {
    name = "My Dashboard Test 1 NEW"
    dashboard_group = "${signalfx_dashboard_group.mydashboardgroup0.id}"

    time_range = "-30m"

    filter {
        property = "collector"
        values = ["cpu", "Diamond"]
    }
    variable {
        property = "region"
        alias = "region"
        values = ["uswest-1-"]
    }
    chart {
        chart_id = "${signalfx_time_chart.mychart0.id}"
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
        chart_id = "${signalfx_text_chart.mynote0.id}"
        width = 12
        height = 1
    }
}
`

func TestAccCreateDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardGroupDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newDashConfig,
				Check:  testAccCheckDashboardGroupResourceExists,
			},
			// Update Everything
			{
				Config: updatedDashConfig,
				Check:  testAccCheckDashboardGroupResourceExists,
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
		case "signalfx_time_chart", "signalfx_list_chart", "signalfx_single_value_chart", "signalfx_heatmap_chart", "signalfx_text_chart":
			chart, err := client.GetChart(rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding time chart %s: %s", rs.Primary.ID, err)
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

	return nil
}

func testAccDashboardGroupDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart", "signalfx_list_chart", "signalfx_single_value_chart", "signalfx_heatmap_chart", "signalfx_text_chart":
			chart, _ := client.GetChart(rs.Primary.ID)
			if chart.Id != "" {
				return fmt.Errorf("Found deleted time chart %s", rs.Primary.ID)
			}
		case "signalfx_dashboard":
			dash, _ := client.GetDashboard(rs.Primary.ID)
			if dash.Id != "" {
				return fmt.Errorf("Found deleted dashboard %s", rs.Primary.ID)
			}
		case "signalfx_dashboard_group":
			dashgroup, _ := client.GetDashboardGroup(rs.Primary.ID)
			if dashgroup.Id != "" {
				return fmt.Errorf("Found deleted dashboard group %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
