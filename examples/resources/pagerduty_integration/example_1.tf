resource "signalfx_pagerduty_integration" "pagerduty_myteam" {
  name    = "PD - My Team"
  enabled = true
  api_key = "1234567890"
}
