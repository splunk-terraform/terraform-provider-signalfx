---
page_title: "Splunk Observability Cloud: signalfx_heatmap_chart"
description: |-
  Allows Terraform to create and manage heat map charts in Splunk Observability Cloud
---

# Resource: signalfx_heatmap_chart

This chart type shows the specified plot in a heat map fashion. This format is similar to the [Infrastructure Navigator](https://signalfx-product-docs.readthedocs-hosted.com/en/latest/built-in-content/infra-nav.html#infra), with squares representing each source for the selected metric, and the color of each square representing the value range of the metric.

## Example

```terraform
resource "signalfx_heatmap_chart" "myheatmapchart0" {
  name = "CPU Total Idle - Heatmap"

  program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

  description = "Very cool Heatmap"

  disable_sampling = true
  sort_by          = "+host"
  group_by         = ["hostname", "host"]
  hide_timestamp   = true
  timezone         = "Europe/Paris"

  color_range {
    min_value = 0
    max_value = 100
    color     = "#ff0000"
  }

  # You can only use one of color_range or color_scale!
  color_scale {
    gte   = 99
    color = "green"
  }
  color_scale {
    lt    = 99 # This ensures terraform recognizes that we cover the range 95-99
    gte   = 95
    color = "yellow"
  }
  color_scale {
    lt    = 95
    color = "red"
  }
}
```

## Arguments

The following arguments are supported in the resource block:

* `name` - (Required) Name of the chart.
* `program_text` - (Required) Signalflow program text for the chart. More info at https://dev.splunk.com/observability/docs/signalflow/.
* `description` - (Optional) Description of the chart.
* `unit_prefix` - (Optional) Must be `"Metric"` or `"Binary`". `"Metric"` by default.
* `minimum_resolution` - (Optional) The minimum resolution (in seconds) to use for computing the underlying program.
* `max_delay` - (Optional) How long (in seconds) to wait for late datapoints.
* `timezone` - (Optional) The property value is a string that denotes the geographic region associated with the time zone, (default UTC).
* `refresh_interval` - (Optional) How often (in seconds) to refresh the values of the heatmap.
* `disable_sampling` - (Optional) If `false`, samples a subset of the output MTS, which improves UI performance. `false` by default.
* `group_by` - (Optional) Properties to group by in the heatmap (in nesting order).
* `sort_by` - (Optional) The property to use when sorting the elements. Must be prepended with `+` for ascending or `-` for descending (e.g. `-foo`).
* `hide_timestamp` - (Optional) Whether to show the timestamp in the chart. `false` by default.
* `color_range` - (Optional, Default) Values and color for the color range. Example: `color_range : { min : 0, max : 100, color : "#0000ff" }`. Look at this [link](https://docs.splunk.com/observability/en/data-visualization/charts/chart-options.html).
  * `min_value` - (Optional) The minimum value within the coloring range.
  * `max_value` - (Optional) The maximum value within the coloring range.
  * `color` - (Required) The color range to use. The starting hex color value for data values in a heatmap chart. Specify the value as a 6-character hexadecimal value preceded by the '#' character, for example "#ea1849" (grass green).
* `color_scale` - (Optional. Conflicts with `color_range`) One to N blocks, each defining a single color range including both the color to display for that range and the borders of the range. Example: `color_scale { gt = 60, color = "blue" } color_scale { lte = 60, color = "yellow" }`. Look at this [link](https://docs.splunk.com/observability/en/data-visualization/charts/chart-options.html).
  * `gt` - (Optional) Indicates the lower threshold non-inclusive value for this range.
  * `gte` - (Optional) Indicates the lower threshold inclusive value for this range.
  * `lt` - (Optional) Indicates the upper threshold non-inclusive value for this range.
  * `lte` - (Optional) Indicates the upper threshold inclusive value for this range.
  * `color` - (Required) The color range to use. Hex values are not supported here. Must be one of red, gold, iris, green, jade, gray, blue, azure, navy, brown, orange, yellow, magenta, cerise, pink, violet, purple, lilac, emerald, chartreuse, yellowgreen, aquamarine.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
