resource "signalfx_customized_auto_detector" "example" {
  parent_id   = "parent-detector"
  name        = "Modified Example Detector"
  description = "This is an example of a modified auto detector resource."
  severity    = "Critical"
  tags        = [
    "tag-01", "tag-02"
  ]
  teams       = [
    "team-01", "team-02"
  ]

  notifications = [
    { type = "Email", email = "example@example.com" },
    { type = "Slack", channel = "#alerts", credential_id = "slack-credential" },
  ]

  inputs = {
    guard = "0.81"
  }

  filters = [
    { key = "service", values = ["web"] },
    { key = "environment", values = ["production"] },
  ]
}
