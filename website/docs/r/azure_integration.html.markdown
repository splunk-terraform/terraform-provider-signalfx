---
layout: "signalfx"
page_title: "SignalFx: signalfx_azure_integration"
sidebar_current: "docs-signalfx-resource-azure-integration"
description: |-
  Allows Terraform to create and manage SignalFx Azure Integrations
---

# Resource: signalfx_azure_integration

SignalFx Azure integrations. For help with this integration see [Monitoring Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure).

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_azure_integration" "azure_myteam" {
  name    = "Azure Foo"
  enabled = true

  resource "signalfx_azure_integration" "azure_myteamXX" {
    name    = "AzureFoo"
    enabled = false

    environment = "azure"

    poll_rate = 300

    secret_key = "XXX"

    app_id = "YYY"

    tenant_id = "ZZZ"

    services = ["microsoft.sql/servers/elasticpools"]

    subscriptions = ["sub-guid-here"]
  }
}
```

## Service Names

~> **NOTE** You can use the data source "signalfx_azure_services" to specify all services.

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `app_id` - (Required) Azure application ID for the SignalFx app. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/getting-started/send-data.html#connect-to-microsoft-azure) in the product documentation.
* `environment` (Optional) What type of Azure integration this is. The allowed values are `\"azure_us_government\"` and `\"azure\"`. Defaults to `\"azure\"`.
* `poll_rate` - (Optional) AWS poll rate (in seconds). One of `60` or `300`.
* `secret_key` - (Required) Azure secret key that associates the SignalFx app in Azure with the Azure tenant ID. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure) in the product documentation.
* `services` - (Required) List of Microsoft Azure service names for the Azure services you want SignalFx to monitor. See the documentation for [Creating Integrations](https://developers.signalfx.com/integrations_reference.html#operation/Create%20Integration) for valida values.
* `subscriptions` - (Required) List of Azure subscriptions that SignalFx should monitor.
* `tenant_id` (Required) Azure ID of the Azure tenant. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure) in the product documentation.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
