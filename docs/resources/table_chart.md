---
page_title: "Splunk Observability Cloud: signalfx_table_chart"
description: |-
  Allows Terraform to create and manage data table charts in Splunk Observability Cloud
---

# Resource: signalfx_table_chart

This special type of chart displays a data table. This table can be grouped by a dimension.

## Example

```terraform
# signalfx_list_chart.Logs-Exec_0:
resource "signalfx_table_chart" "table_0" {
    description             = "beep"
    disable_sampling        = false
    max_delay               = 0
    name                    = "TableChart!"
    program_text            = "A = data('cpu.usage.total').publish(label='CPU Total')"
    group_by                = ["ClusterName"]
}
```

## Arguments

The following arguments are supported in the resource block:

* `name` - (Required) Name of the table chart.
* `program_text` - (Required) The SignalFlow for your Data Table Chart
* `description` - (Optional) Description of the table chart.
* `group_by` - (Optional) Dimension to group by

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
