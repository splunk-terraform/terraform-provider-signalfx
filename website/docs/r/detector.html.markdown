---
layout: "signalfx"
page_title: "SignalFx: signalfx_detector"
sidebar_current: "docs-signalfx-resource-dashboard"
description: |-
  Allows Terraform to create and manage SignalFx Dashboards
---

# Resource: signalfx_detector

Provides a SignalFx detector resource. This can be used to create and manage detectors.

~> **NOTE** If you're interested in using SignalFx detector features such as Historical Anomaly, Resource Running Out, or others then consider building them in the UI first then using the "Show SignalFlow" feature to extract the value for `program_text`. You may also consult the [documentation for detector functions in signalflow-library](https://github.com/signalfx/signalflow-library/tree/master/library/signalfx/detectors).

## Example Usage

```tf
resource "signalfx_detector" "application_delay" {
  count = length(var.clusters)

  name        = " max average delay - ${var.clusters[count.index]}"
  description = "your application is slow - ${var.clusters[count.index]}"
  max_delay   = 30
  tags        = ["app-backend", "staging"]

  # Note that if you use these features, you must use a user's
  # admin key to authenticate the provider, lest Terraform not be able
  # to modify the detector in the future!
  authorized_writer_teams = [signalfx_team.mycoolteam.id]
  authorized_writer_users = ["abc123"]

  program_text = <<-EOF
        signal = data('app.delay', filter('cluster','${var.clusters[count.index]}'), extrapolation='last_value', maxExtrapolations=5).max()
        detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
    EOF
  rule {
    description   = "maximum > 60 for 5m"
    severity      = "Warning"
    detect_label  = "Processing old messages 5m"
    notifications = ["Email,foo-alerts@bar.com"]
  }
  rule {
    description   = "maximum > 60 for 30m"
    severity      = "Critical"
    detect_label  = "Processing old messages 30m"
    notifications = ["Email,foo-alerts@bar.com"]
  }
}

provider "signalfx" {}

variable "clusters" {
  default = ["clusterA", "clusterB"]
}
```

## Notification Format

As SignalFx supports different notification mechanisms a comma-delimited string is used to provide inputs. If you'd like to specify multiple notifications, then each should be a member in the list, like so:

```
notifications = ["Email,foo-alerts@example.com", "Slack,credentialId,channel"]
```

This will likely be changed in a future iteration of the provider. See [SignalFx Docs](https://developers.signalfx.com/detectors_reference.html#operation/Create%20Single%20Detector) for more information. For now, here are some example of how to configure each notification type:

### Email

```
notifications = ["Email,foo-alerts@bar.com"]
```

### Jira

Note that the `credentialId` is the SignalFx-provided ID shown after setting up your Jira integration. (See also `signalfx_jira_integration`.)

```
notifications = ["Jira,credentialId"]
```

### Opsgenie

Note that the `credentialId` is the SignalFx-provided ID shown after setting up your Opsgenie integration. `Team` here is hardcoded as the `responderType` as that is the only acceptable type as per the API docs.

```
notifications = ["Opsgenie,credentialId,responderName,responderId,Team"]
```

### PagerDuty

```
notifications = ["PagerDuty,credentialId"]
```

### Slack

Exclude the `#` on the channel name!

```
notifications = ["Slack,credentialId,channel"]
```

### Team

Sends [notifications to a team](https://docs.signalfx.com/en/latest/managing/teams/team-notifications.html).

```
notifications = ["Team,teamId"]
```

### TeamEmail

Sends an email to every member of a team.

```
notifications = ["TeamEmail,teamId"]
```

### VictorOps

```
notifications = ["VictorOps,credentialId,routingKey"]
```

### Webhook

~> **NOTE** You need to include all the commas even if you only use a credential id below.

You can either configure a Webhook to use an existing integration's credential id:
```
notifications = ["Webhook,credentialId,,"]
```

or configure one inline:
```
notifications = ["Webhook,,secret,url"]
```

## Argument Reference

* `name` - (Required) Name of the detector.
* `program_text` - (Required) Signalflow program text for the detector. More info [in the SignalFx docs](https://developers.signalfx.com/signalflow_analytics/signalflow_overview.html#_signalflow_programming_language).
* `description` - (Optional) Description of the detector.
* `authorized_writer_teams` - (Optional) Team IDs that have write access to this detector. Remember to use an admin's token if using this feature and to include that admin's team id (or user id in `authorized_writer_users`).
* `authorized_writer_users` - (Optional) User IDs that have write access to this detector. Remember to use an admin's token if using this feature and to include that admin's user id (or team id in `authorized_writer_teams`).
* `max_delay` - (Optional) How long (in seconds) to wait for late datapoints. See [Delayed Datapoints](https://signalfx-product-docs.readthedocs-hosted.com/en/latest/charts/chart-builder.html#delayed-datapoints) for more info. Max value is `900` seconds (15 minutes). `Auto` (as little as possible) by default.
* `min_delay` - (Optional) How long (in seconds) to wait even if the datapoints are arriving in a timely fashion. Max value is 900 (15m).
* `show_data_markers` - (Optional) When `true`, markers will be drawn for each datapoint within the visualization. `true` by default.
* `show_event_lines` - (Optional) When `true`, the visualization will display a vertical line for each event trigger. `false` by default.
* `disable_sampling` - (Optional) When `false`, the visualization may sample the output timeseries rather than displaying them all. `false` by default.
* `time_range` - (Optional) Seconds to display in the visualization. This is a rolling range from the current time. Example: `3600` corresponds to `-1h` in web UI. `3600` by default.
* `start_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `end_time` - (Optional) Seconds since epoch. Used for visualization. Conflicts with `time_range`.
* `tags` - (Optional) Tags associated with the detector.
* `teams` - (Optional) Team IDs to associate the detector to.
* `rule` - (Required) Set of rules used for alerting.
    * `detect_label` - (Required) A detect label which matches a detect label within `program_text`.
    * `severity` - (Required) The severity of the rule, must be one of: `"Critical"`, `"Major"`, `"Minor"`, `"Warning"`, `"Info"`.
    * `description` - (Optional) Description for the rule. Displays as the alert condition in the Alert Rules tab of the detector editor in the web UI.
    * `disabled` - (Optional) When true, notifications and events will not be generated for the detect label. `false` by default.
    * `notifications` - (Optional) List of strings specifying where notifications will be sent when an incident occurs. See [Create A Single Detector](https://developers.signalfx.com/detectors_reference.html#operation/Create%20Single%20Detector) for more info.
    * `parameterized_body` - (Optional) Custom notification message body when an alert is triggered. See [Set Up Detectors to Trigger Alerts](https://docs.signalfx.com/en/latest/detect-alert/set-up-detectors.html#about-detectors#alert-settings) for more info.
    * `parameterized_subject` - (Optional) Custom notification message subject when an alert is triggered. See [Set Up Detectors to Trigger Alerts](https://docs.signalfx.com/en/latest/detect-alert/set-up-detectors.html#about-detectors#alert-settings) for more info.
    * `runbook_url` - (Optional) URL of page to consult when an alert is triggered. This can be used with custom notification messages.
    * `tip` - (Optional) Plain text suggested first course of action, such as a command line to execute. This can be used with custom notification messages.
* `viz_options` - (Optional) Plot-level customization options, associated with a publish statement.
    * `label` - (Required) Label used in the publish statement that displays the plot (metric time series data) you want to customize.
    * `display_name` - (Optional) Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.
    * `color` - (Optional) Color to use : gray, blue, azure, navy, brown, orange, yellow, iris, magenta, pink, purple, violet, lilac, emerald, green, aquamarine.
    * `value_unit` - (Optional) A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes). Values values are `Bit, Kilobit, Megabit, Gigabit, Terabit, Petabit, Exabit, Zettabit, Yottabit, Byte, Kibibyte, Mebibyte, Gigibyte, Tebibyte, Pebibyte, Exbibyte, Zebibyte, Yobibyte, Nanosecond, Microsecond, Millisecond, Second, Minute, Hour, Day, Week`.
    * `value_prefix`, `value_suffix` - (Optional) Arbitrary prefix/suffix to display with the value of this plot.

**Notes**

It is highly recommended that you use both `max_delay` in your detector configuration and an `extrapolation` policy in your program text to reduce false positives/negatives.

`max_delay` allows SignalFx to continue with computation if there is a lag in receiving data points.

`extrapolation` allows you to specify how to handle missing data. An extrapolation policy can be added to individual signals by updating the data block in your `program_text`.

See [Delayed Datapoints](https://signalfx-product-docs.readthedocs-hosted.com/en/latest/charts/chart-builder.html#delayed-datapoints) for more info.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the detector.
* `url` - The URL of the detector.

## Import

Detectors can be imported using their string ID (recoverable from URL: `/#/detector/v2/abc123/edit`, e.g.

```
$ terraform import signalfx_detector.application_delay abc123
```
