---
layout: "signalfx"
page_title: "SignalFx: signalfx_dimension_values"
sidebar_current: "docs-signalfx-signalfx-dimension-values"
description: |-
  Provides a list of dimension values given a query
---

# Data Source: signalfx_dimension_values

Use this data source to get a list of dimension values matching the provided query.

~> **NOTE** This data source only allows 1000 values, as it's kinda nuts to make anything with `for_each` that big in SignalFx. This is negotiable.

## Example Usage

```hcl
resource "signalfx_dashboard_group" "mydashboardgroup0" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
}

data "signalfx_dimension_values" "hosts" {
	query = "key:host"
}

resource "signalfx_time_chart" "host_charts" {
		for_each = toset(data.signalfx_dimension_values.hosts.values)

    name = "CPU Total Idle ${each.value}"

		plot_type = "ColumnChart"
		axes_include_zero = true
		color_by = "Metric"

    program_text = <<-EOF
A = data("cpu.idle", filter('host', '${each.key}').publish(label="CPU")
        EOF
}

resource "signalfx_dashboard" "mydashboard1" {
    name = "My Dashboard"
    dashboard_group = signalfx_dashboard_group.mydashboardgroup0.id

    time_range = "-30m"

		grid {
			chart_ids = toset([for v in signalfx_time_chart.host_charts: v.id ])
			width = 3
			height = 1
		}
}
```

## Argument Reference

* `query`

## Attributes Reference

`values` is set to the list of dimension values.
