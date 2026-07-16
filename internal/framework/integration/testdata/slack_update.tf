resource "signalfx_slack_integration" "test" {
  name        = "Updated Slack"
  enabled     = false
  webhook_url = "https://hooks.slack.test/updated"
}
