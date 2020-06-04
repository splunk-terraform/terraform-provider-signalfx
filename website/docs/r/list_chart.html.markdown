---
layout: "signalfx"
page_title: "SignalFx: signalfx_list_chart"
sidebar_current: "docs-signalfx-resource-list-chart"
description: |-
  Allows Terraform to create and manage SignalFx list charts
---

# Resource: signalfx_list_chart

This chart type displays current data values in a list format.

The name of each value in the chart reflects the name of the plot and any associated dimensions. We recommend you click the Pencil icon and give the plot a meaningful name, as in plot B below. Otherwise, just the raw metric name will be displayed on the chart, as in plot A below.


## Example Usage

```tf
resource "signalfx_list_chart" "mylistchart0" {
  name = "CPU Total Idle - List"

  program_text = <<-EOF
    myfilters = filter("cluster_name", "prod") and filter("role", "search")
    data("cpu.total.idle", filter=myfilters).publish()
    EOF

  description = "Very cool List Chart"

  color_by         = "Metric"
  max_delay        = 2
  disable_sampling = true
  refresh_interval = 1

  legend_options_fields {
    property = "collector"
    enabled  = false
  }

  legend_options_fields {
    property = "cluster_name"
    enabled  = true
  }
  legend_options_fields {
    property = "role"
    enabled  = true
  }
  legend_options_fields {
    property = "collector"
    enabled  = false
  }
  legend_options_fields {
    property = "host"
    enabled  = false
  }
  max_precision = 2
  sort_by       = "-value"
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the chart.
* `program_text` - (Required) Signalflow program text for the chart. More info[in the SignalFx docs](https://developers.signalfx.com/signalflow_analytics/signalflow_overview.html#_signalflow_programming_language).
* `description` - (Optional) Description of the chart.
* `unit_prefix` - (Optional) Must be `"Metric"` or `"Binary`". `"Metric"` by default.
* `color_by` - (Optional) Must be one of `"Scale"`, `"Dimension"` or `"Metric"`. `"Dimension"` by default.
* `max_delay` - (Optional) How long (in seconds) to wait for late datapoints.
* `disable_sampling` - (Optional) If `false`, samples a subset of the output MTS, which improves UI performance. `false` by default.
* `refresh_interval` - (Optional) How often (in seconds) to refresh the values of the list.
* `viz_options` - (Optional) Plot-level customization options, associated with a publish statement.
    * `label` - (Required) Label used in the publish statement that displays the plot (metric time series data) you want to customize.
    * `display_name` - (Optional) Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.
    * `color` - (Optional) Color to use : gray, blue, azure, navy, brown, orange, yellow, iris, magenta, pink, purple, violet, lilac, emerald, green, aquamarine.
    * `value_unit` - (Optional) A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes). Values values are `Bit, Kilobit, Megabit, Gigabit, Terabit, Petabit, Exabit, Zettabit, Yottabit, Byte, Kibibyte, Mebibyte, Gigibyte, Tebibyte, Pebibyte, Exbibyte, Zebibyte, Yobibyte, Nanosecond, Microsecond, Millisecond, Second, Minute, Hour, Day, Week`.
    * `value_prefix`, `value_suffix` - (Optional) Arbitrary prefix/suffix to display with the value of this plot.
* `legend_fields_to_hide` - (Optional) List of properties that should not be displayed in the chart legend (i.e. dimension names). All the properties are visible by default. Deprecated, please use `legend_options_fields`.
* `legend_options_fields` - (Optional) List of property names and enabled flags that should be displayed in the data table for the chart, in the order provided. This option cannot be used with `legend_fields_to_hide`.
    * `property` The name of the property to display. Note the special values of `sf_metric` (corresponding with the API's `Plot Name`) which shows the label of the time series `publish()` and `sf_originatingMetric` (corresponding with the API's `metric (sf metric)`) that shows the [name of the metric](https://developers.signalfx.com/signalflow_analytics/functions/data_function.html#table-1-parameter-definitions) for the time series being displayed.
    * `enabled` True or False depending on if you want the property to be shown or hidden.
* `max_precision` - (Optional) Maximum number of digits to display when rounding values up or down.
* `secondary_visualization` - (Optional) The type of secondary visualization. Can be `None`, `Radial`, `Linear`, or `Sparkline`. If unset, the SignalFx default is used (`Sparkline`).
* `color_scale` - (Optional. `color_by` must be `"Scale"`) Single color range including both the color to display for that range and the borders of the range. Example: `[{ gt = 60, color = "blue" }, { lte = 60, color = "yellow" }]`. Look at this [link](https://docs.signalfx.com/en/latest/charts/chart-options-tab.html).
    * `gt` - (Optional) Indicates the lower threshold non-inclusive value for this range.
    * `gte` - (Optional) Indicates the lower threshold inclusive value for this range.
    * `lt` - (Optional) Indicates the upper threshold non-inculsive value for this range.
    * `lte` - (Optional) Indicates the upper threshold inclusive value for this range.
    * `color` - (Required) The color range to use. Must be either gray, blue, navy, orange, yellow, magenta, purple, violet, lilac, green, aquamarine.
* `sort_by` - (Optional) The property to use when sorting the elements. Use `value` if you want to sort by value. Must be prepended with `+` for ascending or `-` for descending (e.g. `-foo`). Note there are some special values for some of the options provided in the UX: `"value"` for Value, `"sf_originatingMetric"` for Metric, and `"sf_metric"` for plot.
* `time_range` - (Optional) How many seconds ago from which to display data. For example, the last hour would be `3600`, etc. Conflicts with `start_time` and `end_time`.
* `start_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `end_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
