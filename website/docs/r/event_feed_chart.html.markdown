---
layout: "signalfx"
page_title: "SignalFx: signalfx_event_feed_chart"
sidebar_current: "docs-signalfx-resource-event-feed-chart"
description: |-
  Allows Terraform to create and manage SignalFx time charts
---

# Resource: signalfx_event_feed_chart

Displays a listing of events as a widget in a dashboard.

## Example Usage

```tf
resource "signalfx_event_feed_chart" "mynote0" {
  name         = "Important Dashboard Note"
  description  = "Lorem ipsum dolor sit amet"
  program_text = "A = events(eventType='Fart Testing').publish(label='A')"

  viz_options {
    label = "A"
    color = "orange"
  }
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the text note.
* `program_text` - (Required) Signalflow program text for the chart. More info[in the SignalFx docs](https://developers.signalfx.com/signalflow_analytics/signalflow_overview.html#_signalflow_programming_language).
* `description` - (Optional) Description of the text note.
* `time_range` - (Optional) From when to display data. SignalFx time syntax (e.g. `"-5m"`, `"-1h"`). Conflicts with `start_time` and `end_time`.
* `start_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `end_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
