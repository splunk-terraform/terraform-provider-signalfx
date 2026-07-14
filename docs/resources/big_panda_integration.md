---
page_title: "Observability Cloud: signalfx_big_panda_integration"
description: |-
  Allows Terraform to create and manage BigPanda Integrations for Splunk Observability Cloud
---
# Resource: signalfx_big_panda_integration

BigPanda integrations. For help with this integration see [Integration with BigPanda](https://docs.splunk.com/Observability/admin/notif-services/bigpanda.html).

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```terraform
resource "signalfx_big_panda_integration" "big_panda_myteam" {
  name    = "BigPanda - My Team"
  enabled = true

  app_key = "my-app-key"
  token   = "my-token"

  # Optional. If omitted, Observability Cloud sends the default BigPanda payload.
  alert_triggered_payload_template = "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"
  alert_resolved_payload_template  = "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"
}
```

## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `app_key` - (Required) Application key you get from BigPanda.
* `token` - (Required) Token you get from BigPanda.
* `alert_triggered_payload_template` - (Optional) A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a triggered notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.
* `alert_resolved_payload_template` - (Optional) A template that Observability Cloud uses to create the BigPanda POST JSON payload when an alert sends a resolved notification to BigPanda. If omitted, Observability Cloud uses the default BigPanda payload.

## Attributes

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
