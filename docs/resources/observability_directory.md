---
page_title: "Splunk Observability Cloud: signalfx_observability_directory"
description: |-
  Manages a safe, path-only Observability Directory entry.
---

# Resource: signalfx_observability_directory

Directory membership and dashboard placement are not managed. Deletion is refused unless the entry is safely empty and provider-manageable.

## Example

```terraform
terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_directory" "dashboards" {
  path   = "teams/platform/dashboards"
  pinned = true
}
```

## Arguments

* `path` - (Required) Decoded logical Directory path.
* `pinned` - (Optional) Whether the Directory entry is pinned. Defaults to `false`.

## Attributes

* `id` - The decoded logical Directory path.
