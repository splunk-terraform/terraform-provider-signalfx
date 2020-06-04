---
layout: "signalfx"
page_title: "SignalFx: signalfx_webhook_resource"
sidebar_current: "docs-signalfx-resource-webhook-integration"
description: |-
  Allows Terraform to create and manage SignalFx Webhook Integrations
---

# Resource: signalfx_webhook_resource

SignalFx Webhook integration.

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_webhook_resource" "webhook_myteam" {
  name          = "Webhook - My Team"
  enabled       = true
  url           = "https://www.example.com"
  shared_secret = "abc1234"

  headers {
    header_key   = "some_header"
    header_value = "value_for_that_header"
  }
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `url` - (Required) The URL to request
* `shared_secret` - (Optional)
* `headers` - (Optional) A header to send with the request
  * `header_key` - (Required) The key of the header to send
  * `header_value` - (Required) The value of the header to send
