---
page_title: "Splunk Observability Cloud: signalfx_detector"
description: |-
  Creates and manages a detector in Splunk Observability Cloud
---

# Resource: signalfx_detector

Provides a Splunk Observability Cloud detector resource.

If you want to use detector features such as Historical Anomaly or Resource Running Out, build the detector in the UI first and use **Show SignalFlow** to obtain its `program_text`. You can also review the [SignalFlow detector library](https://github.com/signalfx/signalflow-library/tree/master/library/signalfx/detectors).

~> **NOTE** To change or remove detector write permissions for another user, authenticate the provider with an administrator's session token. See [Operations that require an administrator session token](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

## Example

```terraform
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

## Enhanced multi-condition detector example

Each `rule.detect_label` must match a label published by a `detect(...).publish('<label>')` statement in `program_text`.

```terraform
resource "signalfx_detector" "enhanced_multi_condition" {
  name        = "Enhanced multi-condition detector"
  description = "Historical anomaly and threshold conditions with custom logic."
  max_delay   = 30
  tags        = ["detectors", "historical-anomaly"]

  program_text = <<-EOF
    from signalfx.detectors.against_periods import conditions

    latency = data('service.latency').mean(by=['service']).publish('service latency')
    error_rate = data('service.error_rate').mean(by=['service']).publish('service error rate')
    saturation = data('service.saturation').mean(by=['service']).publish('service saturation')

    latency_anomaly_fire, latency_anomaly_clear = conditions.mean_std(
      latency,
      window_to_compare=duration('15m'),
      space_between_windows=duration('1w'),
      fire_num_stddev=3,
      clear_num_stddev=2.5,
      orientation='above',
    )

    sustained_errors = when(error_rate > 5, '5m')
    high_saturation = when(saturation > 80, '10m')
    critical_saturation = when(saturation > 95, '5m')

    detect(
      (latency_anomaly_fire and sustained_errors and high_saturation) or critical_saturation,
      latency_anomaly_clear and when(error_rate < 2, '10m') and when(saturation < 70, '10m'),
    ).publish('Historical anomaly and service health')
  EOF

  rule {
    description   = "Historical latency anomaly with elevated error rate and saturation, or critical saturation"
    severity      = "Critical"
    detect_label  = "Historical anomaly and service health"
    notifications = ["Email,foo-alerts@example.com"]
  }
}

provider "signalfx" {}
```

## Notification format

Notifications use the provider's comma-delimited string representation. Multiple destinations are separate list elements:

```terraform
notifications = ["Email,foo-alerts@example.com", "Slack,credentialId,channel"]
```

Supported forms include:

```text
Email,address
Email,address,cc1|cc2,bcc1|bcc2
Jira,credentialId
Opsgenie,credentialId,responderName,responderId,Team
PagerDuty,credentialId
Slack,credentialId,channel
Team,teamId
TeamEmail,teamId
VictorOps,credentialId,routingKey
Webhook,credentialId,,
Webhook,,secret,url
```

Email Cc/Bcc requires the Observability Cloud `emailNotificationCcBccEnabled` organization feature. See the [detector API reference](https://dev.splunk.com/observability/reference/api/detectors/latest) for destination details.

## Delayed datapoints

Use both `max_delay` and an extrapolation policy in `program_text` to reduce false positives and false negatives. `max_delay` allows computation to wait for late datapoints, while extrapolation defines how an individual signal handles missing data. See [Delayed Datapoints](https://docs.splunk.com/observability/en/data-visualization/charts/chart-builder.html#delayed-datapoints).

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the detector.
- `program_text` (String) SignalFlow program text for the detector.

### Optional

- `authorized_writer_teams` (Set of String) Team IDs with write access to the detector.
- `authorized_writer_users` (Set of String) User IDs with write access to the detector.
- `description` (String) Description of the detector.
- `detector_origin` (String) How the detector was created: `Standard` or `AutoDetectCustomization`. Changes replace the resource.
- `disable_sampling` (Boolean) Whether to display all datapoints instead of sampling them.
- `end_time` (Number) End of the absolute visualization range in Unix seconds.
- `max_delay` (Number) Maximum time in seconds to wait for late datapoints.
- `min_delay` (Number) Minimum time in seconds to wait even when datapoints arrive on time.
- `parent_detector_id` (String) Parent AutoDetect detector ID for an AutoDetect customization. Changes replace the resource.
- `rule` (Block Set) Required set of alert rules. (see [below for nested schema](#nestedblock--rule))
- `show_data_markers` (Boolean) Whether to draw markers for datapoints in the visualization.
- `show_event_lines` (Boolean) Whether to draw a vertical line for each triggered event.
- `start_time` (Number) Start of the absolute visualization range in Unix seconds.
- `tags` (Set of String) Tags associated with the detector.
- `teams` (Set of String) Team IDs associated with the detector.
- `time_range` (Number) Relative visualization range in seconds. Defaults to 3600 when absolute times are not configured.
- `timezone` (String) Geographic time zone associated with the detector, for example `Australia/Sydney`.
- `viz_options` (Block Set) Per-publish-label visualization options. (see [below for nested schema](#nestedblock--viz_options))

### Read-Only

- `id` (String) The unique identifier for the resource.
- `label_resolutions` (Map of Number) Resolution in milliseconds for each published detector label.
- `url` (String) URL of the detector in Splunk Observability Cloud.

<a id="nestedblock--rule"></a>
### Nested Schema for `rule`

Required:

- `detect_label` (String) Publish label associated with this alert rule.
- `severity` (String) Severity of the rule.

Optional:

- `description` (String) Description of the rule.
- `disabled` (Boolean) Whether this alert rule is disabled.
- `notifications` (List of String) Ordered comma-delimited notification destinations.
- `parameterized_body` (String) Custom notification body.
- `parameterized_subject` (String) Custom notification subject.
- `reminder_notification` (Block List) Optional repeated-notification settings. (see [below for nested schema](#nestedblock--rule--reminder_notification))
- `runbook_url` (String) Runbook URL for the alert rule.
- `skip_clear_notification_states` (Set of String) Alert clear states that do not send clear notifications.
- `tip` (String) Suggested first course of action.

<a id="nestedblock--rule--reminder_notification"></a>
### Nested Schema for `rule.reminder_notification`

Required:

- `interval_ms` (Number) Notification interval in milliseconds.
- `type` (String) Reminder type. Must be `TIMEOUT`.

Optional:

- `timeout_ms` (Number) Notification timeout in milliseconds.



<a id="nestedblock--viz_options"></a>
### Nested Schema for `viz_options`

Required:

- `label` (String) SignalFlow publish label.

Optional:

- `color` (String) Color name.
- `display_name` (String) Alternate display name.
- `value_prefix` (String) Prefix displayed with plot values.
- `value_suffix` (String) Suffix displayed with plot values.
- `value_unit` (String) Unit attached to the plot values.

## Import

Import a detector using its string ID, which is also present in detector URLs such as `/#/detector/v2/abc123/edit`:

```shell
terraform import signalfx_detector.application_delay abc123
```
