---
layout: "signalfx"
page_title: "SignalFx: signalfx_jira_integration"
sidebar_current: "docs-signalfx-resource-jira-integration"
description: |-
  Allows Terraform to create and manage SignalFx Jira Integrations
---

# Resource: signalfx_jira_integration

SignalFx Jira integrations. For help with this integration see [Integration with Jira](https://docs.signalfx.com/en/latest/admin-guide/integrate-notifications.html#integrate-with-jira).

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```tf
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


## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `auth_method` - (Required) Authentication method used when creating the Jira integration. One of `EmailAndToken` (using `user_email` and `api_token`) or `UsernameAndPassword` (using `username` and `password`).
* `api_token` - (Required if `auth_method` is `EmailAndToken`) The API token for the user email
* `user_email` - (Required if `auth_method` is `EmailAndToken`) Email address used to authenticate the Jira integration.
* `username` - (Required if `auth_method` is `UsernameAndPassword`) User name used to authenticate the Jira integration.
* `password` - (Required if `auth_method` is `UsernameAndPassword`) Password used to authenticate the Jira integration.
* `base_url` - (Required) Base URL of the Jira instance that's integrated with SignalFx.
* `issue_type` - (Required) Issue type (for example, Story) for tickets that Jira creates for detector notifications. SignalFx validates issue types, so you must specify a type that's valid for the Jira project specified in `projectKey`.
* `project_key` - (Required) Jira key of an existing project. When Jira creates a new ticket for a detector notification, the ticket is assigned to this project.
* `assignee_name` - (Required) Jira user name for the assignee.
* `assignee_display_name` - (Optional) Jira display name for the assignee.
