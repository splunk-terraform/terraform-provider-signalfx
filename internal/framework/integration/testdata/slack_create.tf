resource "signalfx_slack_integration" "test" {
  name        = "Primary Slack"
  enabled     = true
  webhook_url = "https://hooks.slack.test/primary"
}
