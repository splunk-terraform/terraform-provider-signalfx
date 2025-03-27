---
page_title: "Splunk Observability Cloud: signalfx_pagerduty_integration"
description: |-
  Allows Terraform to create and manage PagerDuty Integrations for Splunk Observability Cloud
---

# Resource: signalfx_pagerduty_integration

Splunk Observability Cloud PagerDuty integrations.

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```terraform
resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
  name    = "PD - My Team"
  enabled = true
  api_key = "1234567890"
}
```

## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `api_key` - (Required) PagerDuty API key.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
