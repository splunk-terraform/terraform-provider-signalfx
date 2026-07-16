resource "signalfx_team" "test" {
  name        = "Updated Team"
  description = "Updated description"
  members     = ["member-c"]

  notifications_critical = ["Email,updated-critical@example.com"]
  notifications_default  = ["Email,updated-default@example.com"]
  notifications_info     = ["Email,updated-info@example.com"]
  notifications_major    = ["Email,updated-major@example.com"]
  notifications_minor    = ["Email,updated-minor@example.com"]
  notifications_warning  = ["Email,warning@example.com"]
}
