resource "signalfx_pagerduty_integration" "test" {
  name    = "Primary PagerDuty"
  enabled = true
  api_key = "pagerduty-key"
}
