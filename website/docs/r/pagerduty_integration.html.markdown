---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-pagerduty-integration"
description: |-
  Allows Terraform to create and manage SignalFx PagerDuty Integrations
---

# Resource: signalfx_pagerduty_integration

SignalFx PagerDuty integrations

## Example Usage

```terraform
resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
    name = "PD - My Team"
    enabled = true
    api_key = "1234567890"
}
```
## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `api_key` - (Required) PagerDuty API key.
