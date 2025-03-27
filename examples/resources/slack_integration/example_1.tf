resource "signalfx_slack_integration" "slack_myteam" {
  name        = "Slack - My Team"
  enabled     = true
  webhook_url = "http://example.com"
}
