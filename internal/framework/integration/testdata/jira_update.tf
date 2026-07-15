resource "signalfx_jira_integration" "test" {
  name        = "Updated Jira"
  enabled     = false
  auth_method = "EmailAndToken"
  user_email  = "updated@example.test"
  api_token   = "updated-token"

  base_url              = "https://updated.atlassian.test"
  issue_type            = "Problem"
  project_key           = "UPDATED"
  assignee_name         = "updated-assignee"
  assignee_display_name = "Updated Assignee"
}
