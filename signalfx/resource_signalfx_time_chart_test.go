package signalfx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"

	sfx "github.com/signalfx/signalfx-go"
)

const newTimeChartConfig = `
resource "signalfx_time_chart" "mychartXX" {
    name = "CPU Total Idle"
		description = "I am described"

    program_text = <<-EOF
        data('cpu.total.idle').publish(label='CPU Idle')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true

    plot_type = "LineChart"
    show_data_markers = true
		show_event_lines = true
		stacked = false
		axes_precision = 4

    legend_options_fields = [
			{
				property = "collector"
				enabled = false
			}
		]
    viz_options {
        label = "CPU Idle"
        axis = "left"
        color = "orange"
				plot_type = "Histogram"
				value_unit = "Byte"
				value_prefix = "prefix"
				value_suffix = "suffix"
    }

		histogram_options {
			color_theme = "lilac"
		}

    axis_left {
        label = "OMG on fire"
				high_watermark = 2000
				high_watermark_label = "high"
        low_watermark = 1000
				low_watermark_label = "low"
				min_value = 900
				max_value = 2100
    }

		axis_right {
        label = "OMG still on fire"
				high_watermark = 2001
				high_watermark_label = "higher"
        low_watermark = 1001
				low_watermark_label = "lower"
				min_value = 901
				max_value = 2101
    }
}
`

const updatedTimeChartConfig = `
resource "signalfx_time_chart" "mychartXX" {
    name = "CPU Total Idle NEW"
		description = "I am described NEW"

    program_text = <<-EOF
        data('cpu.total.idle').publish(label='CPU Idle')
        EOF

    time_range = 900

		axes_include_zero = true
		unit_prefix = "Binary"
		color_by = "Metric"
		minimum_resolution = 30
		max_delay = 15
		disable_sampling = true

    plot_type = "LineChart"
    show_data_markers = true
		show_event_lines = true
		stacked = false
		axes_precision = 4

    legend_options_fields = [
			{
				property = "collector"
				enabled = false
			}
		]
    viz_options {
        label = "CPU Idle"
        axis = "left"
        color = "orange"
				plot_type = "Histogram"
				value_unit = "Byte"
				value_prefix = "prefix"
				value_suffix = "suffix"
    }

		histogram_options {
			color_theme = "lilac"
		}

    axis_left {
        label = "OMG on fire"
				high_watermark = 2000
				high_watermark_label = "high"
        low_watermark = 1000
				low_watermark_label = "low"
				min_value = 900
				max_value = 2100
    }

		axis_right {
        label = "OMG still on fire"
				high_watermark = 2001
				high_watermark_label = "higher"
        low_watermark = 1001
				low_watermark_label = "lower"
				min_value = 901
				max_value = 2101
    }
}
`

func TestAccCreateTimeChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTimeChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newTimeChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTimeChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "name", "CPU Total Idle"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "description", "I am described"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "program_text", "data('cpu.total.idle').publish(label='CPU Idle')\n"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axes_include_zero", "true"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "unit_prefix", "Binary"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "color_by", "Metric"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "minimum_resolution", "30"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "max_delay", "15"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "disable_sampling", "true"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "time_range", "900"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "show_data_markers", "true"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "show_event_lines", "true"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "stacked", "false"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axes_precision", "4"),

					// Left Axis
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.#", "1"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.high_watermark", "2000"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.high_watermark_label", "high"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.label", "OMG on fire"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.low_watermark", "1000"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.low_watermark_label", "low"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.max_value", "2100"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_left.5950944.min_value", "900"),

					// Right Axis
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.#", "1"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.high_watermark", "2001"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.high_watermark_label", "higher"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.label", "OMG still on fire"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.low_watermark", "1001"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.low_watermark_label", "lower"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.max_value", "2101"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "axis_right.3852869422.min_value", "901"),

					// Viz Options
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.#", "1"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.axis", "left"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.color", "orange"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.label", "CPU Idle"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.plot_type", "Histogram"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.value_prefix", "prefix"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.value_suffix", "suffix"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "viz_options.53506722.value_unit", "Byte"),

					// Legend Options
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "legend_options_fields.#", "1"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "legend_options_fields.0.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "legend_options_fields.0.property", "collector"),
				),
			},
			// Update Everything
			{
				Config: updatedTimeChartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTimeChartResourceExists,
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "name", "CPU Total Idle NEW"),
					resource.TestCheckResourceAttr("signalfx_time_chart.mychartXX", "description", "I am described NEW"),
				),
			},
		},
	})
}

func TestValidatePlotTypeTimeChartAllowed(t *testing.T) {
	for _, value := range []string{"LineChart", "AreaChart", "ColumnChart", "Histogram"} {
		_, errors := validatePlotTypeTimeChart(value, "plot_type")
		assert.Equal(t, len(errors), 0)
	}
}

func TestValidatePlotTypeTimeChartNotAllowed(t *testing.T) {
	_, errors := validatePlotTypeTimeChart("absolute", "plot_type")
	assert.Equal(t, len(errors), 1)
}

func testAccCheckTimeChartResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart":
			chart, err := client.GetChart(rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding chart %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	// Add some time to let the API quiesce. This may be removed in the future.
	time.Sleep(time.Duration(2) * time.Second)

	return nil
}

func testAccTimeChartDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_time_chart":
			chart, _ := client.GetChart(rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("Found deleted chart %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
