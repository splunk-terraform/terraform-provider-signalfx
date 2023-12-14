---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_event_feed_chart"
sidebar_current: "docs-signalfx-resource-event-feed-chart"
description: |-
  Allows Terraform to create and manage time charts in Splunk Observability Cloud
---

# Resource: signalfx_event_feed_chart

Displays a listing of events as a widget in a dashboard.

## Example

```tf
resource "signalfx_event_feed_chart" "mynote0" {
  name         = "Important Dashboard Note"
  description  = "Lorem ipsum dolor sit amet"
  program_text = "A = events(eventType='My Event Type').publish(label='A')"

  viz_options {
    label = "A"
    color = "orange"
  }
}
```

## Arguments

The following arguments are supported in the resource block:

* `name` - (Required) Name of the text note.
* `program_text` - (Required) Signalflow program text for the chart. More info[in the Splunk Observability Cloud docs](https://dev.splunk.com/observability/docs/signalflow/).
* `description` - (Optional) Description of the text note.
* `time_range` - (Optional) From when to display data. Splunk Observability Cloud time syntax (e.g. `"-5m"`, `"-1h"`). Conflicts with `start_time` and `end_time`.
* `start_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `end_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the chart.
* `url` - The URL of the chart.
