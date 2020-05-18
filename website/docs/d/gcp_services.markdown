---
layout: "signalfx"
page_title: "SignalFx: signalfx_gcp_services"
sidebar_current: "docs-signalfx-signalfx-gcp-services"
description: |-
  Provides a list GCP service names.
---

# Data Source: signalfx_gcp_services

Use this data source to get a list of GCP service names.

## Example Usage

```hcl
data "signalfx_gcp_services" "gcp_services" {
}

# Leaves out most of the integration bits, see the docs
# for signalfx_gcp_integration for more
resource "signalfx_gcp_integration" "gcp_myteam" {
   # …

   # All supported services!
   services = data.signalfx_gcp_services.gcp_services.services.*.name
}
```

## Argument Reference

None

## Attributes Reference

`services` is set to all available service names supported by SignalFx
