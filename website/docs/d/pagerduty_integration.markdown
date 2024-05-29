---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_pagerduty_integration"
sidebar_current: "docs-signalfx-signalfx-pagerduty-integration"
description: |-
  Provides information on an existing PagerDuty integration.
---

# Data source: signalfx_pagerduty_integration

Use this data source to get information on an existing PagerDuty integration.

## Example

```hcl
data "signalfx_pagerduty_integration" "pd_integration" {
  name = "PD-Integration"
}
```

## Arguments

* `name` - Specify the exact name of the desired PagerDuty integration

## Attributes

* `id` - The ID of the integration.
* `name` - The name of the integration.
* `enabled` - Whether the integration is enabled.
