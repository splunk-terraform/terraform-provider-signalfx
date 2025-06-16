---
page_title: "Splunk Observability Cloud: signalfx_org_token"
description: |-
  Allows Terraform to create and manage text notes in Splunk Observability Cloud
---

# Resource: signalfx_org_token

Manage Splunk Observability Cloud org tokens.

~> **NOTE** When managing Org tokens, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

## Example

```terraform
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

## Arguments

The following arguments are supported in the resource block:

* `name` - (Required) Name of the token.
* `description` - (Optional) Description of the token.
* `disabled` - (Optional) Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`.
* `secret` - The secret token created by the API. You cannot set this value.
* `store_secret` - (Optional) Whether to store the token's secret in the terraform state. Defaults to `true` for backward compatibility.
* `notifications` - (Optional) Where to send notifications about this token's limits. See the [Notification Format](https://www.terraform.io/docs/providers/signalfx/r/detector.html#notification-format) laid out in detectors.
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
  * `dpm_notification_threshold` - (Optional) DPM level at which Splunk Observability Cloud sends the notification for this token. If you don't specify a notification, Splunk Observability Cloud sends the generic notification.
  * `dpm_limit` - (Required) The datapoints per minute (dpm) limit for this token. If you exceed this limit, Splunk Observability Cloud sends out an alert.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the token.
* `secret` - The assigned token.
