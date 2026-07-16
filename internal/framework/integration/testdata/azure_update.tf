resource "signalfx_azure_integration" "test" {
  name        = "Updated Azure"
  enabled     = false
  environment = "azure_us_government"
  app_id      = "updated-app"
  secret_key  = "updated-secret"
  tenant_id   = "updated-tenant"
  named_token = "ingest-token"

  poll_rate = 600
  services = [
    "microsoft.compute/virtualmachines",
    "microsoft.storage/storageaccounts",
  ]
  additional_services = ["custom/service", "custom/second"]
  subscriptions       = ["subscription-b", "subscription-a"]
  sync_guest_os_namespaces = true
  import_azure_monitor      = false
  use_batch_api             = true

  custom_namespaces_per_service {
    service    = "Microsoft.Compute/virtualMachines"
    namespaces = ["updatedNamespace"]
  }
  custom_namespaces_per_service {
    service    = "Microsoft.Storage/storageAccounts"
    namespaces = ["storageNamespace"]
  }

  resource_filter_rules {
    filter_source = "filter('azure_tag_service', 'payment')"
  }
  resource_filter_rules {
    filter_source = "filter('azure_tag_environment', 'production')"
  }
}
