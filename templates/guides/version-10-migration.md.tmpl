---
page_title: "Migrating to version 10 of the Splunk Observability Cloud provider"
description: |-
  Instructions for configuration or state changes required when migrating resources and data sources to Terraform Plugin Framework in provider version 10.
---

# Migrating to provider version 10

Version 10 migrates provider resources and data sources to Terraform Plugin Framework. Most migrations preserve the existing Terraform type name, configuration, and state. This guide records only changes that require user action.

Each breaking-change section will include old and replacement configuration, automatic state-upgrade behavior, and any required state, import, or recreation commands. Resource and data-source field reference remains generated from the provider schema in the corresponding reference page.

## Provider: remove `custom_app_url`

The deprecated `custom_app_url` provider argument has been removed. Delete it from the provider configuration before upgrading:

```terraform
provider "signalfx" {
  auth_token = var.signalfx_auth_token
  api_url    = "https://api.<realm>.signalfx.com"
}
```

The provider discovers the application URL from the organization API. No resource state changes or import commands are required.

## GCP integration: replace `project_wif_configs`

The deprecated `project_wif_configs` block has been removed from `signalfx_gcp_integration`. Configure Workload Identity Federation with the top-level `workload_identity_federation_config` argument and the `projects` block instead.

Before upgrading, replace configuration such as:

```terraform
project_wif_configs {
  project_id = "example-project"
  wif_config = file("example-project-wif.json")
}
```

with:

```terraform
workload_identity_federation_config = file("wif.json")

projects {
  sync_mode            = "SELECTED"
  selected_project_ids = ["example-project"]
}
```

Apply this configuration with provider version 9 before upgrading where possible. If Terraform reports that the existing state contains the unsupported `project_wif_configs` attribute after upgrading, preserve the integration ID, remove only the Terraform state entry, and import the integration again:

```shell
terraform state rm signalfx_gcp_integration.example
terraform import signalfx_gcp_integration.example <integration-id>
terraform plan
```

Removing the state entry does not delete the integration. Review the final plan before applying it.
