resource "signalfx_pagerduty_integration" "test" {
  name    = "Updated PagerDuty"
  enabled = false
  api_key = "updated-pagerduty-key"
}
