resource "signalfx_jira_integration" "jira_myteamXX" {
  name    = "JiraFoo"
  enabled = false

  auth_method = "UsernameAndPassword"
  username    = "yoosername"
  password    = "paasword"

  # Or…
  #auth_method = "EmailAndToken"
  #user_email = "yoosername@example.com"
  #api_token = "abc123"

  assignee_name         = "testytesterson"
  assignee_display_name = "Testy Testerson"

  base_url    = "https://www.example.com"
  issue_type  = "Story"
  project_key = "TEST"
}
