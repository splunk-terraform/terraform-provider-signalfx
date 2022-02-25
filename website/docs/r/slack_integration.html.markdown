---
layout: "signalfx"
page_title: "SignalFx: signalfx_slack_integration"
sidebar_current: "docs-signalfx-resource-slack-integration"
description: |-
  Allows Terraform to create and manage SignalFx Slack Integrations
---

# Resource: signalfx_slack_integration

SignalFx Slack integration.

~> **NOTE** When managing integrations use a session token for an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator].(https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_slack_integration" "slack_myteam" {
  name        = "Slack - My Team"
  enabled     = true
  webhook_url = "http://example.com"
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `webhook_url` - (Required) Slack incoming webhook URL.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
