---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-slack-integration"
description: |-
  Allows Terraform to create and manage SignalFx Slack Integrations
---

# Resource: signalfx_integration_slack

SignalFx Slack integration.

## Example Usage

```terraform
resource "signalfx_slack_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    webhook_url = "http://example.com"
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `webhook_url` - (Required) Slack incoming webhook URL.
