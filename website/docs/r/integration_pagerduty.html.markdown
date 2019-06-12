---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-integration-pagerduty"
description: |-
  Allows Terraform to create and manage SignalFx PagerDuty Integrations
---

# Resource: signalfx_integration_pagerduty

SignalFx PagerDuty integrations

## Example Usage

```terraform
resource "signalfx_integration" "pagerduty_myteam" {
    name = "PD - My Team"
    enabled = true
    type = "PagerDuty"
    api_key = "1234567890"
}
```
## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `api_key` - (Required) PagerDuty API key.
