---
page_title: "Splunk Observability Cloud: signalfx_time_chart"
description: |-
  Allows Terraform to create and manage time charts in Splunk Observability Cloud
---

# Resource: signalfx_time_chart

Provides a Splunk Observability Cloud time chart resource. This can be used to create and manage the different types of time charts.

Time charts display data points over a period of time.

## Example

```terraform
resource "signalfx_time_chart" "mychart0" {
  name = "CPU Total Idle"

  program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF

  time_range = 3600

  plot_type         = "LineChart"
  show_data_markers = true

  legend_options_fields {
    property = "collector"
    enabled  = false
  }
  legend_options_fields {
    property = "hostname"
    enabled  = false
  }

  viz_options {
    label = "CPU Idle"
    axis  = "left"
    color = "orange"
  }

  axis_left {
    label         = "CPU Total Idle"
    low_watermark = 1000
  }
}
```

## Arguments

The following arguments are supported in the resource block:

* `name` - (Required) Name of the chart.
* `program_text` - (Required) Signalflow program text for the chart. More info [in the Splunk Observability Cloud docs](https://dev.splunk.com/observability/docs/signalflow/).
* `plot_type` - (Optional) The default plot display style for the visualization. Must be `"LineChart"`, `"AreaChart"`, `"ColumnChart"`, or `"Histogram"`. Default: `"LineChart"`.
* `description` - (Optional) Description of the chart.
* `axes_precision` - (Optional) Specifies the digits Splunk Observability Cloud displays for values plotted on the chart. Defaults to `3`.
* `unit_prefix` - (Optional) Must be `"Metric"` or `"Binary`". `"Metric"` by default.
* `color_by` - (Optional) Must be `"Dimension"` or `"Metric"`. `"Dimension"` by default.
* `minimum_resolution` - (Optional) The minimum resolution (in seconds) to use for computing the underlying program.
* `max_delay` - (Optional) How long (in seconds) to wait for late datapoints.
* `timezone` - (Optional) A string denotes the geographic region associated with the time zone.
* `disable_sampling` - (Optional) If `false`, samples a subset of the output MTS, which improves UI performance. `false` by default
* `time_range` - (Optional) How many seconds ago from which to display data. For example, the last hour would be `3600`, etc. Conflicts with `start_time` and `end_time`.
* `start_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `end_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `axes_include_zero` - (Optional) Force the chart to display zero on the y-axes, even if none of the data is near zero.
* `axis_left` - (Optional) Set of axis options.
  * `label` - (Optional) Label of the left axis.
  * `min_value` - (Optional) The minimum value for the left axis.
  * `max_value` - (Optional) The maximum value for the left axis.
  * `high_watermark` - (Optional) A line to draw as a high watermark.
  * `high_watermark_label` - (Optional) A label to attach to the high watermark line.
  * `low_watermark` - (Optional) A line to draw as a low watermark.
  * `low_watermark_label` - (Optional) A label to attach to the low watermark line.
* `axis_right` - (Optional) Set of axis options.
  * `label` - (Optional) Label of the right axis.
  * `min_value` - (Optional) The minimum value for the right axis.
  * `max_value` - (Optional) The maximum value for the right axis.
  * `high_watermark` - (Optional) A line to draw as a high watermark.
  * `high_watermark_label` - (Optional) A label to attach to the high watermark line.
  * `low_watermark` - (Optional) A line to draw as a low watermark.
  * `low_watermark_label` - (Optional) A label to attach to the low watermark line.
* `viz_options` - (Optional) Plot-level customization options, associated with a publish statement.
  * `label` - (Required) Label used in the publish statement that displays the plot (metric time series data) you want to customize.
  * `display_name` - (Optional) Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.
  * `color` - (Optional) Color to use. Must be one of red, gold, iris, green, jade, gray, blue, azure, navy, brown, orange, yellow, magenta, cerise, pink, violet, purple, lilac, emerald, chartreuse, yellowgreen, aquamarine.
  * `axis` - (Optional) Y-axis associated with values for this plot. Must be either `right` or `left`.
  * `plot_type` - (Optional) The visualization style to use. Must be `"LineChart"`, `"AreaChart"`, `"ColumnChart"`, or `"Histogram"`. Chart level `plot_type` by default.
  * `value_unit` - (Optional) A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes). Values values are `Bit, Kilobit, Megabit, Gigabit, Terabit, Petabit, Exabit, Zettabit, Yottabit, Byte, Kibibyte, Mebibyte, Gibibyte (note: this was previously typoed as Gigibyte), Tebibyte, Pebibyte, Exbibyte, Zebibyte, Yobibyte, Nanosecond, Microsecond, Millisecond, Second, Minute, Hour, Day, Week`.
  * `value_prefix`, `value_suffix` - (Optional) Arbitrary prefix/suffix to display with the value of this plot.
* `event_options` - (Optional) Event customization options, associated with a publish statement. You will need to use this to change settings for any `events(…)` statements you use.
  * `label` - (Required) Label used in the publish statement that displays the event query you want to customize.
  * `display_name` - (Optional) Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.
  * `color` - (Optional) Color to use : gray, blue, azure, navy, brown, orange, yellow, iris, magenta, pink, purple, violet, lilac, emerald, green, aquamarine.
* `histogram_options` - (Optional) Only used when `plot_type` is `"Histogram"`. Histogram specific options.
  * `color_theme` - (Optional) Color to use. Must be one of red, gold, iris, green, jade, gray, blue, azure, navy, brown, orange, yellow, magenta, cerise, pink, violet, purple, lilac, emerald, chartreuse, yellowgreen, aquamarine.
* `legend_fields_to_hide` - (Optional) List of properties that should not be displayed in the chart legend (i.e. dimension names). All the properties are visible by default. Deprecated, please use `legend_options_fields`.
* `legend_options_fields` - (Optional) List of property names and enabled flags that should be displayed in the data table for the chart, in the order provided. This option cannot be used with `legend_fields_to_hide`.
  * `property` The name of the property to display. Note the special values of `plot_label` (corresponding with the API's `sf_metric`) which shows the label of the time series `publish()` and `metric` (corresponding with the API's `sf_originatingMetric`) that shows the name of the metric for the time series being displayed.
  * `enabled` True or False depending on if you want the property to be shown or hidden.
* `on_chart_legend_dimension` - (Optional) Dimensions to show in the on-chart legend. On-chart legend is off unless a dimension is specified. Allowed: `"metric"`, `"plot_label"` and any dimension.
* `show_event_lines` - (Optional) Whether vertical highlight lines should be drawn in the visualizations at times when events occurred. `false` by default.
* `show_data_markers` - (Optional) Show markers (circles) for each datapoint used to draw line or area charts. `false` by default.
* `stacked` - (Optional) Whether area and bar charts in the visualization should be stacked. `false` by default.
* `timezone` - (Optional) Time zone that SignalFlow uses as the basis of calendar window transformation methods. For example, if you set "timezone": "Europe/Paris" and then use the transformation sum(cycle="week", cycle_start="Monday") in your chart's SignalFlow program, the calendar window starts on Monday, Paris time. See the [full list of timezones for more](https://dev.splunk.com/observability/docs/signalflow/). `"UTC"` by default.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
