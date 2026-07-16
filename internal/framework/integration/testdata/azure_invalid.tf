resource "signalfx_azure_integration" "test" {
  name         = "Invalid Azure"
  enabled      = true
  environment  = "public"
  app_id       = "app"
  secret_key   = "secret"
  tenant_id    = "tenant"
  poll_rate    = 30
  services     = []
  subscriptions = ["subscription"]
}
