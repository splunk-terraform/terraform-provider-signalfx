resource "signalfx_team" "test" {
  name        = "Primary Team"
  description = "Primary description"
  members     = ["member-a", "member-b"]

  notifications_critical = ["Email,critical@example.com"]
  notifications_default  = ["Webhook,,secret,https://hooks.example.com/default"]
  notifications_info     = ["Slack,slack-id,info-alerts"]
  notifications_major    = ["Team,major-team"]
  notifications_minor    = ["PagerDuty,pagerduty-id"]
  notifications_warning  = ["VictorOps,victor-id,warning"]
}
