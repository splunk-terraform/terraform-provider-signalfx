---
layout: "signalfx"
page_title: "SignalFx: signalfx_dashboard_group"
sidebar_current: "docs-signalfx-resource-dashboard-group"
description: |-
  Allows Terraform to create and manage SignalFx Dashboard Groups
---

# Resource: signalfx_dashboard_group

In the SignalFx web UI, a [dashboard group](https://developers.signalfx.com/dashboard_groups_reference.html) is a collection of dashboards.

~> **NOTE** Dashboard groups cannot be accessed directly, but just via a dashboard contained in them. This is the reason why make show won't show any of yours dashboard groups.

## Example Usage

```tf
resource "signalfx_dashboard_group" "mydashboardgroup0" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  # Note that if you use these features, you must use a user's
  # admin key to authenticate the provider, lest Terraform not be able
  # to modify the dashboard group in the future!
  authorized_writer_teams = [signalfx_team.mycoolteam.id]
  authorized_writer_users = ["abc123"]
}
```

## Example Usage with Permissions

```tf
resource "signalfx_dashboard_group" "mydashboardgroup_withpermissions" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  // You can add up to 25 of entries for permission configurations. 
  // Make sure your account supports this feature!
  permissions {
    principal_id    = "abc123"
    principal_type  = "ORG"
    actions         = ["READ"]
  }
  permissions {
    principal_id    = "abc456"
    principal_type  = "USER"
    actions         = ["READ", "WRITE"]
  }
}
```

## Example Usage With Mirrored Dashboards

```tf
resource "signalfx_dashboard_group" "mydashboardgroup_withmirrors" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"

  // You can add as many of these as you like. Make sure your account
  // supports this feature!
  dashboard {
    dashboard_id         = signalfx_dashboard.gc_dashboard.id
    name_override        = "GC For My Service"
    description_override = "Garbage Collection dashboard maintained by JVM team"

    filter_override {
      property = "service"
      values   = ["myservice"]
      negated  = false
    }

    variable_override {
      property         = "region"
      values           = ["us-west1"]
      values_suggested = ["us-west-1", "us-east-1"]
    }
  }
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `name` - (Required) Name of the dashboard group.
* `description` - (Required) Description of the dashboard group.
* `teams` - (Optional) Team IDs to associate the dashboard group to.
* `authorized_writer_teams` - (Optional) Team IDs that have write access to this dashboard group. Remember to use an admin's token if using this feature and to include that admin's team (or user id in `authorized_writer_teams`). **Note:** Deprecated use `permissions_acl` instead.
* `authorized_writer_users` - (Optional) User IDs that have write access to this dashboard group. Remember to use an admin's token if using this feature and to include that admin's user id (or team id in `authorized_writer_teams`). **Note:** Deprecated use `permissions_acl` instead.
* `permissions_acl` - (Optional) [Permissions](https://docs.splunk.com/Observability/infrastructure/terms-concepts/permissions.html) List of read and write permission configuration to specify which user, team, and organization can view and/or edit your dashboard group. **Note:** This feature is not present in all accounts. Please contact support if you are unsure.
  * `principal_id` - (Required) ID of the user, team, or organization for which you're granting permissions.
  * `principal_type` - (Required) Clarify whether this permission configuration is for a user, a team, or an organization. Value can be one of "USER", "TEAM", or "ORG".
  * `actions` - (Required) Action the user, team, or organization can take with the dashboard group. List of values (value can be "READ" or "WRITE").
* `dashboard` - (Optional) [Mirrored dashboards](https://docs.signalfx.com/en/latest/dashboards/dashboard-mirrors.html) in this dashboard group. **Note:** This feature is not present in all accounts. Please contact support if you are unsure.
  * `dashboard_id` - (Required) The dashboard id to mirror
  * `name_override` - (Optional) The name that will override the original dashboards's name.
  * `description_override` - (Optional) The description that will override the original dashboards's description.
  * `filter_override` - (Optional) The description that will override the original dashboards's description.
    * `property` - (Required) The name of a dimension to filter against.
    * `values` - (Required) A list of values to be used with the `property`, they will be combined via `OR`.
    * `negated` - (Optional) If true,  only data that does not match the specified value of the specified property appear in the event overlay. Defaults to `false`.
  * `filter_override` - (Optional) The description that will override the original dashboards's description.
    * `property` - (Required) A metric time series dimension or property name.
    * `values` - (Optional) (Optional) List of of strings (which will be treated as an OR filter on the property).
    * `values_suggested` - (Optional) A list of strings of suggested values for this variable; these suggestions will receive priority when values are autosuggested for this variable.

## Attributes Reference

In a addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
