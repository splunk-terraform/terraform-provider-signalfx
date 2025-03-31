resource "signalfx_opsgenie_integration" "opgenie_myteam" {
  name    = "Opsgenie - My Team"
  enabled = true
  api_key = "my-key"
  api_url = "https://api.opsgenie.com"
}
