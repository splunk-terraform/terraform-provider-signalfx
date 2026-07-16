resource "signalfx_jira_integration" "test" {
  name        = "Primary Jira"
  enabled     = true
  auth_method = "UsernameAndPassword"
  username    = "primary-user"
  password    = "primary-password"

  base_url              = "https://primary.atlassian.test"
  issue_type            = "Story"
  project_key           = "PRIMARY"
  assignee_name         = "primary-assignee"
  assignee_display_name = "Primary Assignee"
}
