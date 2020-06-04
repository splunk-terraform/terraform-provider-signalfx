---
layout: "signalfx"
page_title: "SignalFx: signalfx_pagerduty_integration"
sidebar_current: "docs-signalfx-resource-pagerduty-integration"
description: |-
  Allows Terraform to create and manage SignalFx PagerDuty Integrations
---

# Resource: signalfx_pagerduty_integration

SignalFx PagerDuty integrations

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
  name    = "PD - My Team"
  enabled = true
  api_key = "1234567890"
}
```
## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `api_key` - (Required) PagerDuty API key.
