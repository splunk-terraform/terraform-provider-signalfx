resource "signalfx_azure_integration" "test" {
  name       = "Primary Azure"
  enabled    = true
  app_id     = "primary-app"
  secret_key = "primary-secret"
  tenant_id  = "primary-tenant"
  named_token = "ingest-token"

  poll_rate    = 120
  services     = ["microsoft.compute/virtualmachines"]
  subscriptions = ["subscription-a"]

  custom_namespaces_per_service {
    service    = "Microsoft.Compute/virtualMachines"
    namespaces = ["customNamespace", "monitoringAgent"]
  }
}
