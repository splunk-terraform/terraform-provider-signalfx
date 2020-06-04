---
layout: "signalfx"
page_title: "SignalFx: signalfx_single_value_chart"
sidebar_current: "docs-signalfx-resource-single-value-chart"
description: |-
  Allows Terraform to create and manage SignalFx single value charts
---

# Resource: signalfx_single_value_chart

This chart type displays a single number in a large font, representing the current value of a single metric on a plot line.

If the time period is in the past, the number represents the value of the metric near the end of the time period.

## Example Usage

```tf
resource "signalfx_single_value_chart" "mysvchart0" {
  name = "CPU Total Idle - Single Value"

  program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

  description = "Very cool Single Value Chart"

  color_by = "Dimension"

  max_delay           = 2
  refresh_interval    = 1
  max_precision       = 2
  is_timestamp_hidden = true
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the chart.
* `program_text` - (Required) Signalflow program text for the chart. More info [in the SignalFx docs](https://developers.signalfx.com/signalflow_analytics/signalflow_overview.html#_signalflow_programming_language).
* `description` - (Optional) Description of the chart.
* `color_by` - (Optional) Must be `"Dimension"`, `"Scale"` or `"Metric"`. `"Dimension"` by default.
* `color_scale` - (Optional. `color_by` must be `"Scale"`) Single color range including both the color to display for that range and the borders of the range. Example: `[{ gt = 60, color = "blue" }, { lte = 60, color = "yellow" }]`. Look at this [link](https://docs.signalfx.com/en/latest/charts/chart-options-tab.html).
    * `gt` - (Optional) Indicates the lower threshold non-inclusive value for this range.
    * `gte` - (Optional) Indicates the lower threshold inclusive value for this range.
    * `lt` - (Optional) Indicates the upper threshold non-inculsive value for this range.
    * `lte` - (Optional) Indicates the upper threshold inclusive value for this range.
    * `color` - (Required) The color range to use. Must be either gray, blue, navy, orange, yellow, magenta, purple, violet, lilac, green, aquamarine.
* `viz_options` - (Optional) Plot-level customization options, associated with a publish statement.
    * `label` - (Required) Label used in the publish statement that displays the plot (metric time series data) you want to customize.
    * `display_name` - (Optional) Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.
    * `color` - (Optional) Color to use : gray, blue, azure, navy, brown, orange, yellow, iris, magenta, pink, purple, violet, lilac, emerald, green, aquamarine.
    * `value_unit` - (Optional) A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes). Values values are `Bit, Kilobit, Megabit, Gigabit, Terabit, Petabit, Exabit, Zettabit, Yottabit, Byte, Kibibyte, Mebibyte, Gigibyte, Tebibyte, Pebibyte, Exbibyte, Zebibyte, Yobibyte, Nanosecond, Microsecond, Millisecond, Second, Minute, Hour, Day, Week`.
    * `value_prefix`, `value_suffix` - (Optional) Arbitrary prefix/suffix to display with the value of this plot.
* `unit_prefix` - (Optional) Must be `"Metric"` or `"Binary"`. `"Metric"` by default.
* `max_delay` - (Optional) How long (in seconds) to wait for late datapoints
* `refresh_interval` - (Optional) How often (in seconds) to refresh the value.
* `max_precision` - (Optional) The maximum precision to for value displayed.
* `is_timestamp_hidden` - (Optional) Whether to hide the timestamp in the chart. `false` by default.
* `secondary_visualization` - (Optional) The type of secondary visualization. Can be `None`, `Radial`, `Linear`, or `Sparkline`. If unset, the SignalFx default is used (`None`).
* `show_spark_line` - (Optional) Whether to show a trend line below the current value. `false` by default.
