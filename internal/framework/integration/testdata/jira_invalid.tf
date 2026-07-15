resource "signalfx_jira_integration" "test" {
  name          = "Invalid Jira"
  enabled       = true
  auth_method   = "OAuth"
  username      = "user"
  password      = "password"
  user_email    = "user@example.test"
  api_token     = "token"
  base_url      = "https://invalid.atlassian.test"
  issue_type    = "Story"
  project_key   = "INVALID"
  assignee_name = "assignee"
}
