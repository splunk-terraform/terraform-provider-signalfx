resource "signalfx_customized_auto_detector" "example" {
  parent_id   = "parent-detector"
  name        = "Modified Example Detector"
  description = "This is an example of a modified auto detector resource."
  severity    = "Critical"
  tags        = []
  teams       = []

  notifications = [
    { type = "Email", email = "example@example.com" },
    { type = "Slack", channel = "#alerts", credential_id = "slack-credential" },
  ]

  inputs = {
    guardrail = "0.81"
  }

  filters = [
    { key = "service", values = ["web"] },
    { key = "environment", values = ["production"] },
  ]
}
