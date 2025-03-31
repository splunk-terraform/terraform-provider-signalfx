---
page_title: "Splunk Observability Cloud: signalfx_slack_integration"
description: |-
  Allows Terraform to create and manage Slack Integrations for Splunk Observability Cloud
---

# Resource: signalfx_slack_integration

Slack integration.

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```terraform
resource "signalfx_slack_integration" "slack_myteam" {
  name        = "Slack - My Team"
  enabled     = true
  webhook_url = "http://example.com"
}
```

## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `webhook_url` - (Required) Slack incoming webhook URL.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
