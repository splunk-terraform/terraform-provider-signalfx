resource "signalfx_team" "myteam0" {
  name        = "Best Team Ever"
  description = "Super great team no jerks definitely"

  members = [
    "userid1",
    "userid2",
    # …
  ]

  notifications_critical = [
    "PagerDuty,credentialId"
  ]

  notifications_info = [
    "Email,notify@example.com"
  ]
}
