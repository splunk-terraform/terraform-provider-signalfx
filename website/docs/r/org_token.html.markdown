---
layout: "signalfx"
page_title: "SignalFx: signalfx_org_token"
sidebar_current: "docs-signalfx-resource-org-token"
description: |-
  Allows Terraform to create and manage SignalFx text notes
---

# Resource: signalfx_org_token

Manage SignalFx org tokens.

## Example Usage

```tf
resource "signalfx_org_token" "myteamkey0" {
  name          = "TeamIDKey"
  description   = "My team's rad key"
  notifications = ["Email,foo-alerts@bar.com"]

  host_or_usage_limits {
    host_limit                              = 100
    host_notification_threshold             = 90
    container_limit                         = 200
    container_notification_threshold        = 180
    custom_metrics_limit                    = 1000
    custom_metrics_notification_threshold   = 900
    high_res_metrics_limit                  = 1000
    high_res_metrics_notification_threshold = 900
  }
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the token.
* `description` - (Optional) Description of the token.
* `disabled` - (Optional) Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`.
* `secret` - The secret token created by the API. You cannot set this value.
* `notifications` - (Optional) Where to send notifications about this token's limits. Please consult the [Notification Format](https://www.terraform.io/docs/providers/signalfx/r/detector.html#notification-format) laid out in detectors.
* `host_or_usage_limits` - (Optional) Specify Usage-based limits for this token.
  * `host_limit` - (Optional) Max number of hosts that can use this token
  * `host_notification_threshold` - (Optional) Notification threshold for hosts
  * `container_limit` - (Optional) Max number of Docker containers that can use this token
  * `container_notification_threshold` - (Optional) Notification threshold for Docker containers
  * `custom_metrics_limit` - (Optional) Max number of custom metrics that can be sent with this token
  * `custom_metrics_notification_threshold` - (Optional) Notification threshold for custom metrics
  * `high_res_metrics_limit` - (Optional) Max number of hi-res metrics that can be sent with this toke
  * `high_res_metrics_notification_threshold` - (Optional) Notification threshold for hi-res metrics
* `dpm_limits` (Optional) Specify DPM-based limits for this token.
  * `dpm_notification_threshold` - (Optional) DPM level at which SignalFx sends the notification for this token. If you don't specify a notification, SignalFx sends the generic notification.
  * `dpm_limit` - (Required) The datapoints per minute (dpm) limit for this token. If you exceed this limit, SignalFx sends out an alert.
