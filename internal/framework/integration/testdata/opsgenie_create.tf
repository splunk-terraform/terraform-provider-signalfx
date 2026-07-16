resource "signalfx_opsgenie_integration" "test" {
  name    = "Primary Opsgenie"
  enabled = true
  api_key = "secret-key"
  api_url = "https://api.opsgenie.com"
}
