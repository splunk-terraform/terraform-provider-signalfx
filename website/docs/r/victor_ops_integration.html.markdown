---
layout: "signalfx"
page_title: "SignalFx: signalfx_victor_ops_integration"
sidebar_current: "docs-signalfx-resource-victor-ops-integration"
description: |-
  Allows Terraform to create and manage SignalFx VictorOps Integrations
---

# Resource: signalfx_victor_ops_integration

SignalFx VictorOps integration.

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_victor_ops_integration" "vioctor_ops_myteam" {
  name     = "VictorOps - My Team"
  enabled  = true
  post_url = "https://alert.victorops.com/integrations/generic/1234/alert/$key/$routing_key"
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `post_url` - (Optional) Victor Ops REST API URL.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
