---
layout: "signalfx"
page_title: "SignalFx: signalfx_azure_integration"
sidebar_current: "docs-signalfx-resource-azure-integration"
description: |-
Allows Terraform to create and manage SignalFx Azure Integrations
---

# Resource: signalfx_azure_integration

SignalFx Azure integrations. For help with this integration see [Monitoring Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure).

~> **NOTE** When managing integrations use a session token for an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_azure_integration" "azure_myteam" {
  name    = "Azure Foo"
  enabled = true

  environment = "azure"

  poll_rate = 300

  secret_key = "XXX"

  app_id = "YYY"

  tenant_id = "ZZZ"

  services = ["microsoft.sql/servers/elasticpools"]

  subscriptions = ["sub-guid-here"]

  # Optional
  additional_services = ["some/service", "another/service"]

  # Optional
  custom_namespaces_per_service {
    service = "Microsoft.Compute/virtualMachines"
    namespaces = [ "monitoringAgent", "customNamespace" ]
  }

  # Optional
  resource_filter_rules {
    filter_source = "filter('azure_tag_service', 'payment') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"
  }
  resource_filter_rules {
    filter_source = "filter('azure_tag_service', 'notification') and (filter('azure_tag_env', 'prod-us') or filter('azure_tag_env', 'prod-eu'))"
  }
}
```

## Service Names

~> **NOTE** You can use the data source "signalfx_azure_services" to specify all services.

## Argument Reference

* `app_id` - (Required) Azure application ID for the SignalFx app. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/getting-started/send-data.html#connect-to-microsoft-azure) in the product documentation.
* `enabled` - (Required) Whether the integration is enabled.
* `custom_namespaces_per_service` - (Optional) Allows for more fine-grained control of syncing of custom namespaces, should the boolean convenience parameter `sync_guest_os_namespaces` be not enough. The customer may specify a map of services to custom namespaces. If they do so, for each service which is a key in this map, we will attempt to sync metrics from namespaces in the value list in addition to the default namespaces.
  * `namespaces` - (Required) The additional namespaces.
  * `service` - (Required) The name of the service.
* `environment` (Optional) What type of Azure integration this is. The allowed values are `\"azure_us_government\"` and `\"azure\"`. Defaults to `\"azure\"`.
* `name` - (Required) Name of the integration.
* `named_token` - (Optional) Name of the org token to be used for data ingestion. If not specified then default access token is used.
* `poll_rate` - (Optional) Azure poll rate (in seconds). Value between `60` and `600`. Default: `300`.
* `resource_filter_rules` - (Optional) List of rules for filtering Azure resources by their tags. 
  * `filter_source` - (Required) Expression that selects the data that SignalFx should sync for the resource associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function. The source of each filter rule must be in the form filter('key', 'value'). You can join multiple filter statements using the and and or operators. Referenced keys are limited to tags and must start with the azure_tag_ prefix.
* `secret_key` - (Required) Azure secret key that associates the SignalFx app in Azure with the Azure tenant ID. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure) in the product documentation.
* `services` - (Required) List of Microsoft Azure service names for the Azure services you want SignalFx to monitor. See the documentation for [Creating Integrations](https://developers.signalfx.com/integrations_reference.html#operation/Create%20Integration) for valida values.
* `subscriptions` - (Required) List of Azure subscriptions that SignalFx should monitor.
* `sync_guest_os_namespaces` - (Optional) If enabled, SignalFx will try to sync additional namespaces for VMs (including VMs in scale sets): telegraf/mem, telegraf/cpu, azure.vm.windows.guest (these are namespaces recommended by Azure when enabling their Diagnostic Extension). If there are no metrics there, no new datapoints will be ingested. Defaults to false.
* `import_azure_monitor` - (Optional) If enabled, SignalFx will sync also Azure Monitor data. If disabled, SignalFx will import only metadata. Defaults to true.
* `tenant_id` (Required) Azure ID of the Azure tenant. To learn how to get this ID, see the topic [Connect to Microsoft Azure](https://docs.signalfx.com/en/latest/integrations/azure-info.html#connect-to-azure) in the product documentation.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
