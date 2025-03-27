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
