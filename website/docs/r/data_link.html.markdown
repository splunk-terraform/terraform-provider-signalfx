---
layout: "signalfx"
page_title: "SignalFx: signalfx_data_link"
sidebar_current: "docs-signalfx-resource-data-link"
description: |-
  Allows Terraform to create and manage SignalFx data link
---

# Resource: signalfx_org_token

Manage SignalFx [Data Links](https://docs.signalfx.com/en/latest/managing/data-links.html).

## Example Usage

```terraform
# A global link to SignalFx dashboard.
resource "signalfx_data_link" "my_data_link" {
    property_name = "pname"
    property_value = "pvalue"

    target_signalfx_dashboard {
      is_default = true
      name = "sfx_dash"
			dashboard_group_id = "${signalfx_dashboard_group.mydashboardgroup0.id}"
			dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    }
}

# A dashboard-specific link to an external URL
resource "signalfx_data_link" "my_data_link_dash" {
		dashboard_id = "${signalfx_dashboard.mydashboard0.id}"
    property_name = "pname2"
    property_value = "pvalue"

    target_external_url {
			is_default = false
      name = "ex_url"
      time_format = "ISO8601"
      url = "https://www.example.com"
      property_key_mapping = {
        foo = "bar"
      }
    }
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `property_name` - (Optional) Name (key) of the metadata that's the trigger of a data link. If you specify `property_value`, you must specify `property_name`.
* `property_value` - (Optional) Value of the metadata that's the trigger of a data link. If you specify this property, you must also specify `property_name`.
* `dashboard_id` - (Optional) If provided, scopes this data link to the supplied dashobard id. If omitted then the link will be global.
* `target_external_url` - (Optional) Link to an external URL
* `target_signalfx_dashboard` - (Optional) Link to a SignalFx dashboard
  * `name` (Required) User-assigned target name. Use this value to differentiate between the link targets for a data link object.
  * `dashboard_id` - (Required) SignalFx-assigned ID of the dashboard link target
  * `dashboard_group_id` - (Required) SignalFx-assigned ID of the dashboard link target's dashboard group
  * `is_default` - (Optional) Flag that designates a target as the default for a data link object. `true` by default
