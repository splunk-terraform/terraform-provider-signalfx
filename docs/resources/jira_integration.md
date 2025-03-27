---
page_title: "Splunk Observability Cloud: signalfx_jira_integration"
description: |-
  Allows Terraform to create and manage Jira Integrations for Splunk Observability Cloud
---

# Resource: signalfx_jira_integration

Splunk Observability Cloud Jira integrations. For help with this integration see [Integration with Jira](https://docs.splunk.com/observability/en/admin/notif-services/jira.html).

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```terraform
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
```

## Arguments

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `auth_method` - (Required) Authentication method used when creating the Jira integration. One of `EmailAndToken` (using `user_email` and `api_token`) or `UsernameAndPassword` (using `username` and `password`).
* `api_token` - (Required if `auth_method` is `EmailAndToken`) The API token for the user email
* `user_email` - (Required if `auth_method` is `EmailAndToken`) Email address used to authenticate the Jira integration.
* `username` - (Required if `auth_method` is `UsernameAndPassword`) User name used to authenticate the Jira integration.
* `password` - (Required if `auth_method` is `UsernameAndPassword`) Password used to authenticate the Jira integration.
* `base_url` - (Required) Base URL of the Jira instance that's integrated with SignalFx.
* `issue_type` - (Required) Issue type (for example, Story) for tickets that Jira creates for detector notifications. Splunk Observability Cloud validates issue types, so you must specify a type that's valid for the Jira project specified in `projectKey`.
* `project_key` - (Required) Jira key of an existing project. When Jira creates a new ticket for a detector notification, the ticket is assigned to this project.
* `assignee_name` - (Required) Jira user name for the assignee.
* `assignee_display_name` - (Optional) Jira display name for the assignee.

## Attributes

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
