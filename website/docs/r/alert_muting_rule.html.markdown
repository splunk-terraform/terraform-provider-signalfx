---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_alert_muting_rule"
sidebar_current: "docs-signalfx-resource-alert-muting-rule"
description: |-
  Allows Terraform to create and manage Splunk Observability Cloud Alert Muting Rules
---

# Resource: signalfx_alert_muting_rule

Provides a Splunk Observability Cloud resource for managing alert muting rules. See [Mute Notifications](https://docs.splunk.com/Observability/alerts-detectors-notifications/mute-notifications.html) for more information.

Splunk Observability Cloud currently allows linking an alert muting rule with only one detector ID. Specifying multiple detector IDs makes the muting rule obsolete.

~> **WARNING** Splunk Observability Cloud does not allow the start time of a **currently active** muting rule to be modified. Attempting to modify a currently active rule destroys the existing rule and creates a new rule. This might result in the emission of notifications.

## Example

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

## Arguments

* `description` - (Required) The description for this muting rule
* `start_time` - (Required) Starting time of an alert muting rule as a Unit time stamp in seconds.
* `stop_time` - (Optional) Stop time of an alert muting rule as a Unix time stamp in seconds.
* `detectors` - (Optional) A convenience attribute that associated this muting rule with specific detector IDs. Currently, only one ID is supported.
* `filter` - (Optional) Filters for this rule. See [Creating muting rules from scratch](https://docs.splunk.com/Observability/alerts-detectors-notifications/mute-notifications.html#rule-from-scratch) for more information.
  * `property` - (Required) The property to filter.
  * `property_value` - (Required) The property value to filter.
  * `negated` - (Optional) Determines if this is a "not" filter. Defaults to `false`.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the alert muting rule.
* `effective_start_time`
