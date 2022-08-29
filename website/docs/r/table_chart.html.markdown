---
layout: "signalfx"
page_title: "SignalFx: signalfx_table_chart"
sidebar_current: "docs-signalfx-resource-table-chart"
description: |-
  Allows Terraform to create and manage SignalFx Data Table Charts
---

# Resource: signalfx_table_chart

This special type of chart displays a Data Table. This Table can be grouped by a Dimension.

## Example Usage

```tf
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

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the text note.
* `program_text` - (Required) The SignalFlow for your Data Table Chart
* `description` - (Optional) Description of the text note.
* `group_by` - (Optional) Dimension to group by

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
