provider "signalfx" {

}

resource "signalfx_team" "my_team" {
  name        = "test"
  description = "My awesome team that includes all my friends"

  members = [
    "m001",
    "m002",
  ]

  notifications_default  = ["Email,example@localhost"]
  notifications_info     = ["Email,example@localhost"]
  notifications_minor    = ["Email,example@localhost"]
  notifications_warning  = ["Email,example@localhost"]
  notifications_major    = ["Email,example@localhost"]
  notifications_critical = ["Email,example@localhost"]
}