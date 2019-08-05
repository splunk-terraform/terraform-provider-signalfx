---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-org-token"
description: |-
  Allows Terraform to create and manage SignalFx text notes
---

# Resource: signalfx_org_token

Manage SignalFx org tokens.

## Example Usage

```terraform
resource "signalfx_org_token" "myteamkey0" {
    name = "TeamIDKey"
    description = "My team's rad key"
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the token.
* `description` - (Optional) Description of the token.
* `disabled` - (Optional) Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`.
