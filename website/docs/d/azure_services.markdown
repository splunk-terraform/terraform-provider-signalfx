---
layout: "signalfx"
page_title: "SignalFx: signalfx_azure_services"
sidebar_current: "docs-signalfx-signalfx-azure-services"
description: |-
  Provides a list Azure service names.
---

# Data Source: signalfx_azure_services

Use this data source to get a list of Azure service names.

## Example Usage

```hcl
data "signalfx_azure_services" "azure_services" {
}

# Leaves out most of the integration bits, see the docs
# for signalfx_azure_integration for more
resource "signalfx_azure_integration" "azure_myteam" {
   # …

   # All supported services!
   services = data.signalfx_azure_services.azure_services.services.*.name
}
```

## Argument Reference

None

## Attributes Reference

`services` is set to all available service names supported by SignalFx
