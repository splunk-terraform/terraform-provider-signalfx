---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_single_value_chart"
sidebar_current: "docs-signalfx-resource-single-value-chart"
description: |-
  Allows Terraform to create and manage single value charts in Splunk Observability Cloud
---

# Resource: signalfx_slo_chart

This chart type displays an overview of your SLO and can give more specific insights into your SLI performance using different filter and customized time ranges.

## Example

```tf
resource "signalfx_slo_chart" "myslochart0" {
  slo_id = "GbOHXbSAEAA"
}
```

## Arguments

The following arguments are supported in the resource block:

* `slo_id` - (Required) ID of SLO object.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
