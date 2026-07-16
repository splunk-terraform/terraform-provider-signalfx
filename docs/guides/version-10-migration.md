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

## AWS integration: token is sensitive and empty regions are rejected during planning

The `token` argument on `signalfx_aws_integration` is now marked sensitive. If an output or another value exposes it, Terraform might require that destination to be marked `sensitive = true`.

The resource now rejects an empty `regions` set during planning instead of waiting until apply. Update invalid configurations before upgrading; valid external-ID and security-token configurations do not need state migration.

## Organization token: remove the preview flag and validate DPM limits during planning

The `signalfx_org_token` behavior previously available through the `vnext.org-token` feature preview is now the only implementation. Remove that preview entry before upgrading:

```terraform
provider "signalfx" {
  feature_preview = {
    # Remove: "vnext.org-token" = true
  }
}
```

The `dpm_limit` and `dpm_notification_threshold` values now use the signed 32-bit range required by the Splunk Observability Cloud API. Values outside `-2147483648` through `2147483647` are rejected during planning instead of being truncated during apply.

The limit blocks are represented as single nested blocks instead of SDK set values. Configuration syntax does not change. Terraform normally reconciles the existing state automatically; if an existing token with `host_or_usage_limits` or `dpm_limits` reports an incompatible state representation, preserve the token name and reimport it:

```shell
terraform state rm signalfx_org_token.example
terraform import signalfx_org_token.example <token-name>
terraform plan
```

Removing the state entry does not delete the organization token. Tokens without either limit block require no state migration.

## Dimension values: limits above 1,000 are now honored

`signalfx_dimension_values` now retrieves multiple API pages when `limit` is greater than 1,000. Earlier provider versions accepted values through 10,000 but accidentally returned at most 1,000 results.

If downstream resources use the returned `values` with `for_each`, upgrading can add instances when more than 1,000 dimensions match. Set `limit = 1000` before upgrading to retain the previous effective cap, or review the expanded plan before applying it.

## Automated archival settings: ruleset limits are validated during planning

The `ruleset_limit` argument on `signalfx_automated_archival_settings` now uses the signed 32-bit range required by the Splunk Observability Cloud API. Values outside `-2147483648` through `2147483647` are rejected during planning instead of during apply. Valid settings require no configuration or state migration.

## Automated archival exemptions: each resource now reads only its managed metrics

`signalfx_automated_archival_exempt_metric` now filters the organization-wide exemption list using the comma-delimited metric IDs stored for that Terraform resource. Earlier provider versions assigned every organization exemption to every instance of the resource, so multiple resources and imported subsets could absorb metrics they did not manage.

The configuration block shape is unchanged. Existing resources whose state contains only their own metrics require no migration. If a plan removes unrelated `exempt_metrics` entries that were previously absorbed into state, review the plan and apply it; the provider will retain only the IDs owned by that resource. If the state ID is missing or does not identify the intended metrics, reimport the intended comma-delimited API IDs before applying:

```shell
terraform state rm signalfx_automated_archival_exempt_metric.example
terraform import signalfx_automated_archival_exempt_metric.example <metric-id-1>,<metric-id-2>
terraform plan
```

Removing the state entry does not delete the exempt metrics. Changes to the `exempt_metrics` blocks continue to replace the resource.

## Metric rulesets: singleton blocks and timestamps use native Framework types

`signalfx_metric_ruleset` now represents `matcher`, `aggregator`, `restoration`, and `routing_rule` as single nested blocks. Their HCL block syntax is unchanged, but configurations that supplied more than one of these blocks must retain only the block that was intended; earlier versions silently used the first set element.

The `created`, `last_updated`, `restoration.start_time`, and `restoration.stop_time` fields now use native 64-bit numbers. Numeric literals and numeric string literals are normally coerced automatically. If a restoration timestamp comes from a string-typed variable or expression and Terraform reports a type error, convert it explicitly:

```terraform
restoration {
  start_time = tonumber(var.restoration_start_time)
  stop_time  = tonumber(var.restoration_stop_time)
}
```

Framework normally reconciles the singleton state representation automatically. If an existing ruleset reports an incompatible state shape, preserve its API ID and reimport it:

```shell
terraform state rm signalfx_metric_ruleset.example
terraform import signalfx_metric_ruleset.example <ruleset-id>
terraform plan
```

Removing the state entry does not delete the metric ruleset. Start-only restoration jobs are now retained in state, invalid timestamps return plan diagnostics instead of panicking, and `last_updated_by_name` is now populated when returned by the API.

## Alert muting rules: recurrence values are validated during planning

`signalfx_alert_muting_rule` keeps the existing `filter` and `recurrence` block syntax. The `recurrence.value` field now accepts only the positive signed 32-bit range required by the Splunk Observability Cloud API (`1` through `2147483647`). Earlier provider versions could silently overflow larger values during apply; update an out-of-range value before upgrading.

The Framework implementation retains set semantics for `filter` and `recurrence` so it remains compatible with the protocol-5 mux required by the chart and dashboard resources that remain on SDKv2. Terraform normally reconciles the existing set state automatically. If Terraform reports an incompatible state representation, preserve the API ID and reimport the muting rule:

```shell
terraform state rm signalfx_alert_muting_rule.example
terraform import signalfx_alert_muting_rule.example <muting-rule-id>
terraform plan
```

Removing the state entry does not delete the alert muting rule. The configured `start_time` and `stop_time` remain seconds, while the read-only `effective_start_time` remains the API timestamp in milliseconds.

## Detectors: absolute times are normalized and immutable ancestry replaces the resource

`signalfx_detector` keeps the existing HCL block shapes: `rule` and `viz_options` remain sets, `reminder_notification` remains a maximum-one list block, notification destinations remain ordered strings, and `skip_clear_notification_states` remains a set.

The Framework implementation now consistently treats `start_time`, `end_time`, and `time_range` as seconds. Earlier SDKv2 reads accidentally wrote absolute API timestamps in milliseconds into `start_time` and `end_time`. The first refresh after upgrading can therefore correct those state values by a factor of 1,000 without changing the remote detector. Review the first plan. If legacy state cannot be reconciled, preserve the detector ID and reimport it:

```shell
terraform state rm signalfx_detector.example
terraform import signalfx_detector.example <detector-id>
terraform plan
```

Removing the state entry does not delete the detector. Absolute `end_time` now requires `start_time`; absolute and relative time fields remain mutually exclusive. Time values are limited to the range that can be safely converted to API milliseconds, detector delays remain limited to 0–900 seconds, and reminder intervals/timeouts reject negative values during planning.

`detector_origin` and `parent_detector_id` are creation-time API properties. Changing either now plans replacement instead of sending an update that the API cannot reliably honor. An `AutoDetectCustomization` detector must provide `parent_detector_id`.

Provider-level default tags and teams are still added to detector API requests. Terraform state retains the detector-specific configured values so provider defaults do not create perpetual plans. An imported detector has no configuration from which to distinguish provider defaults, so its initial imported state contains every tag and team returned by the API; remove unwanted entries from configuration after import and review the resulting plan.

## SLOs: target and alert-rule constraints are validated during planning

**signalfx_slo** keeps its existing ordered block syntax for **input**, **target**, **alert_rule**, **rule**, **parameters**, and **reminder_notification**. Existing valid configurations do not need an HCL rewrite.

The Framework implementation now validates target-dependent fields before calling the API. A **RollingWindow** target requires **compliance_period** and cannot configure calendar fields. A **CalendarWindow** target requires **cycle_type** and cannot configure **compliance_period**. Alert-rule types must be unique, every alert-rule group must contain a rule, and every SLO must include a **BREACH** group. Negative reminder intervals and timeouts are also rejected during planning.

The API can supply default values for SLO alert parameters. Those defaults now materialize in Terraform state only when the corresponding **parameters** block is present in configuration, or when the SLO is imported. Omitting the block still uses API defaults remotely but keeps the configured state null, preventing default-only state churn. Review the first plan after upgrading if downstream expressions read unconfigured parameter defaults from state.

## Charts and dashboards remain on SDKv2

Version 10 continues to serve chart and dashboard product types through the muxed SDKv2 provider. This boundary includes dashboard, dashboard group, data link, event feed chart, heatmap chart, list chart, log timeline, log view, single value chart, SLO chart, table chart, text chart, and time chart resources. Their Terraform type names, state, import behavior, and configuration remain unchanged by the Framework migration.

Deprecations belonging exclusively to these deferred products are retained in version 10: dashboard and dashboard-group legacy permission fields, list/time-chart **legend_fields_to_hide**, and time-chart **tags**. They will require their own documented breaking cleanup when the chart and dashboard migration resumes. No action is required for them as part of the current migration.
