resource "signalfx_dashboard_group" "mydashboardgroup0" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  # Note that if you use these features, you must use a user's
  # admin key to authenticate the provider, lest Terraform not be able
  # to modify the dashboard group in the future!
  authorized_writer_teams = [signalfx_team.mycoolteam.id]
  authorized_writer_users = ["abc123"]
}
