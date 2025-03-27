---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_victor_ops_integration"
sidebar_current: "docs-signalfx-resource-victor-ops-integration"
description: |-
  Allows Terraform to create and manage Splunk On-Call Integrations
---

# Resource: signalfx_victor_ops_integration

Splunk On-Call integrations.

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```hcl
resource "signalfx_victor_ops_integration" "vioctor_ops_myteam" {
  name     = "Splunk On-Call - My Team"
  enabled  = true
  post_url = "https://alert.victorops.com/integrations/generic/1234/alert/$key/$routing_key"
}
```

## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `post_url` - (Optional) Splunk On-Call REST API URL.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
