provider "signalfx" {}

resource "signalfx_team" "example_test" {
  provider = signalfx

  name        = "my team"
  description = "An example of team"

  notifications_critical = ["Email,test@example.com"]
  notifications_default  = ["Webhook,,secret,https://www.example.com"]
  notifications_info     = ["Webhook,,secret,https://www.example.com/2"]
  notifications_major    = ["Webhook,,secret,https://www.example.com/3"]
  notifications_minor    = ["Webhook,,secret,https://www.example.com/4"]
  notifications_warning  = ["Webhook,,secret,https://www.example.com/5"]
}
