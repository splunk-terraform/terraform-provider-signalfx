---
layout: "signalfx"
page_title: "SignalFx: signalfx_alert_muting_rule"
sidebar_current: "docs-signalfx-resource-alert-muting-rule"
description: |-
  Allows Terraform to create and manage SignalFx Alert Muting Rules
---

# Resource: signalfx_alert_muting_rule

Provides a SignalFx resource for managing alert muting rules. See [Mute Notifications](https://docs.signalfx.com/en/latest/detect-alert/mute-notifications.html) for more information.

~> **WARNING** SignalFx does not allow the start time of a **currently active** muting rule to be modified. As such, attempting to modify a currently active rule will destroy the existing rule and create a new rule. This may result in the emission of notifications.

## Example Usage

```tf
resource "signalfx_alert_muting_rule" "rool_mooter_one" {
  description = "mooted it NEW"

  start_time = 1573063243
  stop_time  = 0 # Defaults to 0

  detectors = [signalfx_detector.some_detector_id]

  filter {
    property       = "foo"
    property_value = "bar"
  }
}
```

## Argument Reference

* `description` - (Required) The description for this muting rule
* `start_time` - (Required) Starting time of an alert muting rule as a Unit time stamp in seconds.
* `stop_time` - (Optional) Starting time of an alert muting rule as a Unix time stamp in seconds.
* `detectors` - (Optional) A convenience attribute that associated this muting rule with specific detector ids.
* `filter` - (Optional) Filters for this rule. See [Creating muting rules from scratch](https://docs.signalfx.com/en/latest/detect-alert/mute-notifications.html#rule-from-scratch) for more information.
  * `property` - (Required) The property to filter.
  * `property_value` - (Required) The property value to filter.
  * `negated` - (Optional) Determines if this is a "not" filter. Defaults to `false`.
