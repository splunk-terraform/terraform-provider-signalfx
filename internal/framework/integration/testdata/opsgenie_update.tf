resource "signalfx_opsgenie_integration" "test" {
  name    = "Updated Opsgenie"
  enabled = false
  api_key = "updated-secret-key"
  api_url = "https://api.eu.opsgenie.com"
}
