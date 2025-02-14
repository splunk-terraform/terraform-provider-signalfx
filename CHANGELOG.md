## Unreleased

## 9.7.2

BUGFIXES:

* Delete AWS integration when the resource_signalfx_aws_integration is destroyed https://github.com/splunk-terraform/terraform-provider-signalfx/pull/566.

## 9.7.1

BUGFIXES:

* Revert "Add validation of viz objects (#554)" https://github.com/splunk-terraform/terraform-provider-signalfx/pull/564
  
## 9.7.0

IMPROVEMENTS:

* Adding support for collecting only recommended stats https://github.com/splunk-terraform/terraform-provider-signalfx/pull/563
* Add validation of viz objects https://github.com/splunk-terraform/terraform-provider-signalfx/pull/554

## 9.6.1

BUGFIXES:

* Fix creation and update of Team notification policies. [#560](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/560)

## 9.3.0

IMPROVEMENTS:

* Added support for Workload Identity Federation auth method in GCP integration https://github.com/splunk-terraform/terraform-provider-signalfx/pull/546.

## 9.2.1

BUGFIXES:

* documentation: Update metric_ruleset.html.markdown https://github.com/splunk-terraform/terraform-provider-signalfx/pull/501 to fix indentation

## 9.2.0

IMPROVEMENTS:

* `signalfx_metric_ruleset`: Updated restoration under Archived Metrics Exception Rules
  * Add optional `stopTime` field and update `startTime` as a required field [#496](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/496)-

## 9.1.7

IMPROVEMENTS:

* `signalfx_metric_ruleset`: Updated `routing_rule` field to support routing to `Achived` destination.
  * Added optional `exception_rules` field to support exception rules when a metric is routed to `Archived` [#491](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/491)

## 9.1.6

IMPROVEMENTS:

* `signalfx_detector`: added an optional fields: `detector_origin` and `parent_detector_id` - to enable the creation of custom AutoDetect detectors via terraform. [#490](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/490)

## 9.1.5

BUGFIXES:
* Fix an issue with removing all teams/tags/authorized users/authorized teams from detector resource [#489](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/489)

## 9.1.4

IMPROVEMENTS:

* `signalfx_alert_muting_rule`: added an optional field: `recurrence` [#480](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/480)
* `signalfx_webhook_integration`: added optional fields: `method`, `payloadTemplate` [#479](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/479)

## 9.1.3

IMPROVEMENTS:
* Add an option to specify calendar targets for SLO [#476](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/476)

## 9.1.2

BUGFIXES:
* Fix HTTP retry [#467](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/467) [#468](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/468)

## 9.1.1

IMPROVEMENTS:
* Small correction in slo docs [#466](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/466)

## 9.1.0

IMPROVEMENTS:
* Add resource/resource_signalfx_slo to support creating SLO via terraform [#463](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/463)

## 9.0.1

IMPROVEMENTS:
* Updated documentation with correct names and terminology. [#458](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/458)
* Style and grammar improvements in the docs. [#458](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/458)

BUGFIXES:
* Fixed broken links in the documentation. [#458](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/458)

## 9.0.0
BREAKING CHANGES:
* `signalfx_aws_integration` resource: The `regions` field is now required and no longer accepts an empty value.
  Regions must be now declared explicitly to avoid unexpected increased cost whenever a new region is enabled
  on the AWS account [#450](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/450)

## 8.1.0
IMPROVEMENTS:
* Update documentation for `signalfx_gcp_integration` [#447](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/447)
* Add Logs Timeline Chart resource [#448](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/448)

## 8.0.0
IMPROVEMENTS:
* Remove AWS/Azure/GCP service (a.k.a. namespace) validation to make TF provider more flexible 
  and rely on the Splunk Observability Cloud API validation logic. [#444](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/444)
* Remove data resources signalfx_aws_services, signalfx_azure_services, signalfx_gcp_services 
  as they were based on a no longer maintained lists in the signalfx-go library. Users may use 
  empty list to specify "all services" instead or use strings to specify selected services. 
* Fix external-id field handling when token based AWS integration is created in two separate
  Terraform runs. [#445](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/445)
* Update documentation for `signalfx_aws_integration` [#446](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/446)

## 7.0.0
BUGFIXES:
* Fixes issue with calculating correct height of a row when using grid or column layout in dashboard

IMPROVEMENTS:
* Upgrade terraform SDK to v2 [#437](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/437)
* Remove deprecated fields: `use_get_metric_data_method` in AWS integration resource and `whitelist` in GCP integration resource. [#430](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/430)
* Upgrade signalfx-go to add new Azure services. [#436](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/436)

BREAKING CHANGES:
* `signalfx_azure_integration` resource: The `resource_filter_rules.filter.source` field has been
  replaced with `resource_filter_rules.filter_source`, which is the string filter source itself.
  This change is due to the previous implementation of the `filter` and `source` fields using
  the wrong types -- the upgrade to the v2 SDK has stricter validation that required it to be changed.
* `signalfx_metric_ruleset` resource: The `routing_rule` field has been changed to a set type that
  requires the removal of the `=` after `routing_rule` -- the upgrade to the v2 SDK has stricter
  validation that required it to be changed.

## 6.24.0
BUGFIXES:
* Fixes backwards incompatibility for GCP integration resource introduced in `6.21.0` [#429](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/429)

IMPROVEMENTS:
* Deprecate `use_get_metric_data_method` in AWS integration resource [#428](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/428)

## 6.23.0
WARNING: This version is NOT backward compatible for GCP integration resource. This is fixed in 6.24.0

IMPROVEMENTS:
* provider: Use go-retryablehttp for automatic retries with exponential backoff. Add `retry_max_attempts` (default=4), `retry_wait_min_seconds` (default=1), and `retry_wait_max_seconds` (default=30) configuration options. [#416](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/416)
* resource/signalfx_metric_ruleset: added along with docs

## 6.22.0
WARNING: This version is NOT backward compatible for GCP integration resource. This is fixed in 6.24.0

BUGFIXES
* resource/signalfx_table_chart: Added `color` option to `viz_options` to fix `Error: Invalid address to set: []string{"viz_options", "0", "color"}` [#410](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/410)

## 6.21.0
WARNING: This version is NOT backward compatible for GCP integration resource. This is fixed in 6.24.0

IMPROVEMENTS:
* resource/signalfx_gcp_integration: Add `custom_metric_type_domains` option and rename deprecated field `whitelist` to `include_list` [#412](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/412)

BUGFIXES:
* remove empty dashboards in tfstate when non-mirrored dashboards are included in a dashboard group [#413](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/413)

## 6.20.0
 IMPROVEMENTS:
 * resource/signalfx_table_chart: Added `viz_options` option [#402](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/402)

## 6.19.0
IMPROVEMENTS:
* data_source_azure_services: add new services [#403](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/403)

BUGFIXES:
* remove validation with list of supported AWS namespaces in `metric_stats_to_sync` [#400](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/400)

## 6.18.0
IMPROVEMENTS:
* resource/signalfx_gcp_integration: Add `import_gcp_metrics` option [#397](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/397)

BUGFIXES:
* resource/signalfx_gcp_integration: Fix `import_azure_monitor` option - setting it as false now disables metrics properly [#397](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/397)

## 6.17.0
IMPROVEMENTS:
* resource/signalfx_table_chart added along with docs [#389](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/389)

BUGFIXES:
* resource/resource_signalfx_org_token: Persist token even when token limits is not defined [#392](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/392)

## 6.16.0
IMPROVEMENTS:
* resource/signalfx_logs_list_chart renamed to resource/signalfx_log_view: 
* Added docs for resource/signalfx_log_view [#387](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/387)

## 6.15.0
IMPROVEMENTS:
* resource/signalfx_logs_list_chart: Added a new resource for logs list chart [#383](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/383)


## 6.14.0

IMPROVEMENTS:
* resource/signalfx_gcp_integration: add `use_metric_source_project_for_quota` field to the GCP integration [#382](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/382)
* resource/signalfx_slack_integration: fixed documentation for the signalfx_webhook_integration resource [#377](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/377)

## 6.13.1

BUGFIXES:
* resource/detector: Fixes regression after changing `teams` field type from `List` to `Set` [#376](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/376)

## 6.13.0

BUGFIXES:
* resource/dashboard: Fix documentation for  `permissions` field [#373](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/373)
* resource/dashboard_group: Fix documentation for  `permissions` field [#373](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/373)
* resource/detector: Fix issue with `teams` ordering [#372](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/372)
* upgrade signalfx-go to fix issue with `show_data_markers` field on detector resource [#371](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/371)

## 6.12.1
IMPROVEMENTS:
* resource/signalfx_aws_integration: documentation update (metric_stats_to_sync parameter) [#369](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/369)

## 6.12.0

IMPROVEMENTS:
* resource/signalfx_aws_integration: Add `sync_custom_namespaces_only` option [#366](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/366)
* resource/signalfx_aws_integration: documentation update (sorted arguments alphabetically) [#368](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/368)
* resource/signalfx_azure_integration: documentation update (described named_token parameter, sorted arguments alphabetically) [#368](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/368)
* resource/signalfx_gcp_integration: documentation update (described named_token parameter, sorted arguments alphabetically) [#368](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/368)

## 6.11.1
IMPROVEMENTS: 
* resource/signalfx_aws_integration: documentation update (described named_token parameter) [#367](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/367)
 
## 6.11.0

IMPROVEMENTS:
* resource/signalfx_aws_integration: add `metricStatsToSync` support [#362](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/362)
* resource/dashboard: Added `permissions` field, deprecation notice for `authorized_writer_teams` and `authorized_writer_users`  [#357](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/357)
* resource/dashboard_group: Added `permissions` field, deprecation notice for `authorized_writer_teams` and `authorized_writer_users` [#357](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/357)
* resource/signalfx_service_now_integration: add support for ServiceNow integration [#361](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/361)
* resource/signalfx_azure_integration: Add `additional_services` and `resource_filter_rules` [#359](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/359)
* resource/signalfx_aws_integration: documentation update (AWS IAM policy update) [#356](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/356)
* resource/signalfx_aws_integration, resource/signalfx_azure_integration, resource/signalfx_gcp_integration: Align `poll_rate` with Splunk Observability API [#355](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/355)

BUGFIXES:
* resource/signalfx_gcp_integration: fix broken links in documentation [#358](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/358)
* resource/signalfx_alert_muting_rule: allow destroying the resource when muting is in the past [#360](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/360)
* resource/signalfx_alert_muting_rule: fix `detectors` property [#360](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/360)

## 6.10.0

IMPROVEMENTS:
* enable support for darwin_arm64 architecture  [#347](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/347)
* resource/signalfx_detector: add `label_resolutions` support  [#338](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/338)
* resource/signalfx_dashboard: add `tags` support [#349](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/349)
* resource/signalfx_aws_integration: refactor internal implementation [#354](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/354)
* documentation editorial updates  [#346](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/346)

## 6.9.0

IMPROVEMENTS:
* resource/signalfx_aws_integration: add support for AWS metric streams and AWS logs synchronization [#351](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/351) [#353](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/353)

## 6.8.1

BUGFIXES:
* resource/signalfx_org_token: Fix `dpm_limits` and add `auth_scopes` schema option [#342](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/342)

## 6.8.0

IMPROVEMENTS:
* resource/signalfx_detector: Add a CustomDiff to validate the `programText` along with the `rule` attribute at Plan stage [#337](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/337)

## 6.7.10

BUGFIXES:
* data_source_aws_services, data_source_azure_services, data_source_gcp_services: add new services [#328](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/328)
* resource/signalfx_dashboard_group: Teams in dashboard groups should be an unordered set rather than a list [#331](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/331)
* resource/signalfx_alert_muting_rule: Support new type of filter value [#333](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/333)

## 6.7.9

BUGFIXES:
* resource/signalfx_dashboard_group: Teams should not impose ordering on dashboard_groups by using list [#329](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/329)

## 6.7.8

BUGFIXES:
* resource/signalfx_dashboard_group: Correctly handle mirrored dashboards [#319](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/319)

## 6.7.7

BUGFIXES:
* resource/aws_integration: Fix AWS namespaces [#314](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/314) and [#315](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/315)
* resource/signalfx_detector: Remove unnecessary sleep [#313](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/313)

## 6.7.6

BUGFIXES:
* resource/signalfx_data_link: Handle extraneous empty targets [#309](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/309)
* resource/signalfx_detector: Convert tags from list to set and fix set field [#306](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/306)

## 6.7.5

BUGFIXES
* resource/signalfx_aws_integration: Allow specifying a poll rate for AWS integration to up to 10 minutes. [#307](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/307)

## 6.7.4

BUGFIXES
* resource/signalfx_aws_integration: Custom namespaces listed using `custom_namespace_sync_rule` will now correctly emit metrics. [#300](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/300)

## 6.7.3

IMPROVEMENTS:
* Upgrade signalfx-go to get better base URL handling with paths

## 6.7.2

IMPROVEMENTS:
* data_source/azure_services: Update signalfx-go to add missing azure services (see [related PR](https://github.com/signalfx/signalfx-go/pull/123)).

## 6.7.1

IMPROVEMENTS:
* data_source/pagerduty: Update signalfx-go to fix query parameter issue and add enabled flag.

## 6.7.0

IMPROVEMENTS:
* resource/signalfx_data_link: Remove is_default from unsupported targets [#267](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/267)

## 6.6.0

IMPROVEMENTS:
* resource/signalfx_detector: Added `TimeZone` argument. [#285](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/285)

## 6.5.0

IMPROVEMENTS:
* resource/signalfx_detector: Added tags argument. [#283](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/283)

BUFIXES:
* data/signalfx_gcp_services: Fixed GCP data provider [#282](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/282)

## 6.4.0

IMPROVEMENTS:
* resource/list_chart: Added `timezone` argument to set Calendar Window Time Zone in the chart.
* resource/heat_map_chart: Added `timezone` argument to set Calendar Window Time Zone in the chart.

## 6.3.0 (December 21, 2020)

IMPROVEMENTS:
* resource/detector: Add `min_delay` argument.

FEATURES:
* provider: Added data source `signalfx_pagerduty_integration`. [#274](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/274)

## 6.2.0 (December 7, 2020)

IMPROVEMENTS:
* resource/single_value_chart: Added `timezone` argument to set Calendar Window Time Zone in the chart.

## 6.1.0 (November 6, 2020)

IMPROVEMENTS:
* resource/list_chart: Added `hideMissingValues` argument to show or hide missing values in the chart.

## 6.0.0 (October 23, 2020)

IMPROVEMENTS:
* resource/detector: Added back old method for setting teams.
* resource/dashboard_group: Added back old method for setting teams.

BREAKING CHANGES:
* resource/team: Removed short-lived method of setting detectors and dashboard_groups on team object.

## 5.0.2 (October 23, 2020)

BUGFIXES:
* provider: Fix nil panic due to nil user in configuration method ([#260](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/260)

## 5.0.1 (September 23, 2020)

BUGFIXES:
* resource/dashboard_group: The field `import_qualifiers` would not resolve to a clean plan if the dashboard group had an entry like:

    ```json
    "importQualifiers" : [ {
      "filters" : [ ],
      "metric" : ""
    } ]
    ```

  With this change the plan will at least be clean when the empty resource is included in tf:

    ```
    import_qualifiers {
    }
    ```

    This can be removed by sending a manual API request to update the dashboard group by setting `importQualifiers: []`. However if you modify the dashboard group in the UI the empty importQualifiers entry will return.

## 5.0.0 (September 10, 2020)

BREAKING CHANGES:
* resource/dashboard_group: The field `teams` have been removed, please use the `team` resource's `dashboard_groups` argument. [#244](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/244)
* resource/detector: The field `teams` has been removed, please use the `team` resource's `detectors` argument. [#244](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/244)

IMPROVEMENTS:
* resource/team: The new arguments `detectors` and `dashboard_groups` have been added. [#244](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/244)

## 4.26.4 (August 11, 2020)

IMPROVEMENTS:
* resource/dashboard: Document `authorized_writer_teams` and `authorized_writer_users` options. [#239](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/239)
* provider: User-Agent has been reverted back to the older, more information version. [#240](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/240)

## 4.26.3 (August 10, 2020)

BUGFIXES:
* resource/detector: Only "set" a start/end time when there isn't a time range. Fixes conflicting options on import of detectors. [#238](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/238)

## 4.26.2 (August 10, 2020)

IMPROVEMENTS:
* provider: Bump Terraform SDK to v1.15.0. [#237](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/237)

## 4.26.1 (August 8, 2020)

BUGFIXES:
* provider: Removing a description from a chart now properly unsets that description, fixing unclean plans. [#236](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/236)

## 4.26.0 (August 8, 2020)

FEATURES:
* resource/aws_integration: Add `enable_check_large_volume` option. [#234](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/234)

IMPROVEMENTS:
* resource/aws_integration: Allow `poll_rate` to be a range from 60 to 300. [#234](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/234)

## 4.25.0 (August 4, 2020)

BUGFIXES:
* resource/aws_integration: Moved `named_token` to `signalfx_aws_integration`. [#231](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/231)

## 4.24.0 (July 28, 2020)

FEATURES:
* resource/azure_integration: Added `custom_namespaces_per_service` and `sync_guest_os_namespaces`. [#226](https://github.com/signalfx/terraform-provider-signalfx/pull/226)

## 4.23.3 (July 27, 2020)

IMPROVEMENTS:
* provider: Improved documentation to reflect all available color choices. [#223](https://github.com/terraform-providers/terraform-provider-signalfx/pull/223)
* provider: Add goreleaser bits and move to new organization for Terraform 0.13 compatability. [#225](https://github.com/splunk-terraform/terraform-provider-signalfx/pull/225)

## 4.23.2 (June 26, 2020)

IMPROVEMENTS:
* provider: Adjusted the HTTP User-Agent supplied by the provider when making API calls. [#221](https://github.com/terraform-providers/terraform-provider-signalfx/pull/221)

## 4.23.1 (June 10, 2020)

IMPROVEMENTS:
* provider: Bumped signalfx-go dependency which requires the use of `context.Context` in many client calls. No material change otherwise.
* provider: Various doc improvements around formatting, syntax, and more. Thanks [@pdecat](https://github.com/pdecat)! [#217](https://github.com/terraform-providers/terraform-provider-signalfx/pull/217)
* provider/detector: Now sets the `packageSpecifications` field to an empty string, which is an API requirement for some advanced program text use cases. [#220](https://github.com/terraform-providers/terraform-provider-signalfx/pull/220)

## 4.23.0 (June 02, 2020)

IMPROVEMENTS:
* provider: AWS, Azure, and GCP integrations now have an undocumented `named_token` argument. [#214](https://github.com/terraform-providers/terraform-provider-signalfx/pull/214)

## 4.22.0 (May 29, 2020)

IMPROVEMENTS:
* provider: HTTP logging is now enabled in debug mode. Thanks [@pdecat](https://github.com/pdecat)! [#211](https://github.com/terraform-providers/terraform-provider-signalfx/pull/211)
* provider: Allow API URL and Custom App URL to be set from environment variables. Thanks [@pdecat](https://github.com/pdecat)! [#213](https://github.com/terraform-providers/terraform-provider-signalfx/pull/213)
* resource/gcp_integration: Add support for compute metadata whitelist. Thanks [@pdecat](https://github.com/pdecat)! [#212](https://github.com/terraform-providers/terraform-provider-signalfx/pull/212)

## 4.21.0 (May 18, 2020)

IMPROVEMENTS:
* provider: Added `signalfx_gcp_services` data source. [#207](https://github.com/terraform-providers/terraform-provider-signalfx/pull/207/)
* resource/aws_integration: Adjusted validation of poll rate to use SDK validator. [#207](https://github.com/terraform-providers/terraform-provider-signalfx/pull/207/)
* resource/azure_integration: Adjusted validation of poll rate and environment to use SDK validators. [#207](https://github.com/terraform-providers/terraform-provider-signalfx/pull/207/)
* resource/gcp_integration: Adjusted validation of poll rate to use SDK validator. [#207](https://github.com/terraform-providers/terraform-provider-signalfx/pull/207/)

## 4.20.1 (May 12, 2020)

BUGFIXES:
* provider/azure_integration: Fixed some typos in Azure service names. [#205](https://github.com/terraform-providers/terraform-provider-signalfx/pull/205)

## 4.20.0 (May 11, 2020)

IMPROVEMENTS:
* provider: Added data sources `signalfx_aws_services` and `signalfx_azure_services` such that managing AWS and Azure integrations that use "all" services is a bit easier. [#204](https://github.com/terraform-providers/terraform-provider-signalfx/pull/204)
* provider/azure_integration: Updated the list of Azure services. [#203](https://github.com/terraform-providers/terraform-provider-signalfx/pull/203)

## 4.19.7 (May 07, 2020)

IMPROVEMENTS:

* provider: Various resources now ensure that `program_text` is not too big or too small. [#201](https://github.com/terraform-providers/terraform-provider-signalfx/pull/201)

## 4.19.6 (May 06, 2020)

IMPROVEMENTS:
* provider: Bump version of Terraform SDK to older version. [#200](https://github.com/terraform-providers/terraform-provider-signalfx/pull/200)
* provider: Fixed a problem with a test case wherein data links were colliding. [#200](https://github.com/terraform-providers/terraform-provider-signalfx/pull/200)
* provider: Add `timeout_seconds` configuration option and default it to 120, up from 30. [#200](https://github.com/terraform-providers/terraform-provider-signalfx/pull/200)

## 4.19.5 (April 29, 2020)

IMPROVEMENTS:
* provider: Some additional checks to ensure HTTP cleanliness, hopefully preventing possible hangs or leaks. [#198](https://github.com/terraform-providers/terraform-provider-signalfx/pull/198)

## 4.19.4 (April 24, 2020)

BUGFIXES:
* resource/heatmap_chart: Importing some heatmaps would fail. Fixed by defaulting to an empty `color_range` if none is specified. [#196](https://github.com/terraform-providers/terraform-provider-signalfx/pull/196)

## 4.19.3 (April 22, 2020)

IMPROVEMENTS:
* resource/org_token: The field `secret` is now available on the token. [#194](https://github.com/terraform-providers/terraform-provider-signalfx/pull/194)

## 4.19.2 (April 22, 2020)

BUGFIXES:
* resource/org_token: No longer crashes when trying to create without any limits. [#192](https://github.com/terraform-providers/terraform-provider-signalfx/pull/192)

## 4.19.1 (April 13, 2020)

IMPROVEMENTS:
* provider: Now includes a user-agent in it's requests. [#190](https://github.com/terraform-providers/terraform-provider-signalfx/pull/190)
* provider: Bump various dependencies, including the Terraform SDK [#190](https://github.com/terraform-providers/terraform-provider-signalfx/pull/190)

## 4.19.0 (April 13, 2020)

BUGFIXES:
* resource/signalfx_team: Fix a spelling error. Thanks [@ajwood-acquia](https://github.com/ajwood-acquia) [#185](https://github.com/terraform-providers/terraform-provider-signalfx/pull/185)
* resource/signalfx_list_chart: Correct confusing documentation around `legend_options_fields` and it's `property` attribute. [@ebutleratlassian](https://github.com/ebutleratlassian) [#187](https://github.com/terraform-providers/terraform-provider-signalfx/pull/187)
* resource/signalfx_azure_integration: The `services` property is now required and must have at least one item in it. [#189](https://github.com/terraform-providers/terraform-provider-signalfx/pull/189)

IMPROVEMENTS:
* resource/signalfx_heatmap_chart: Improve `color_scale` example and fix indentation. Thanks [@ebutleratlassian](https://github.com/ebutleratlassian) [#186](https://github.com/terraform-providers/terraform-provider-signalfx/pull/186)

FEATURES:
* provider: Added data source `signalfx_dimension_values`. [#188](https://github.com/terraform-providers/terraform-provider-signalfx/pull/188)

## 4.18.6 (March 27, 2020)

IMPROVEMENTS:

* provider: Added support for proxies via Go's [ProxyFromEnvironment](https://golang.org/pkg/net/http/#ProxyFromEnvironment). Thanks [RafeKettler](https://github.com/RafeKettler)! [#183](https://github.com/terraform-providers/terraform-provider-signalfx/pull/183)

## 4.18.5 (March 26, 2020)

BUGFIXES:

* resource/aws_integration: Using `services` no longer generates unclean plans when there are no changes. [#180](https://github.com/terraform-providers/terraform-provider-signalfx/pull/180)

## 4.18.4 (March 20, 2020)

IMPROVEMENTS:

* provider: There are now timeouts on HTTP connection (5s), TLS negotiation (5s) and overall HTTP transaction (30s) durations to avoid long blocks and generate errors where needed. [#177](https://github.com/terraform-providers/terraform-provider-signalfx/pull/177)

BUGFIXES:

* resource/data_link: Setting `is_default` was having no effect and causing unclean plans. [#176](https://github.com/terraform-providers/terraform-provider-signalfx/pull/176)

## 4.18.3 (March 16, 2020)

IMPROVEMENTS:

* resource/aws_integration: Added some missing AWS services. [#173](https://github.com/terraform-providers/terraform-provider-signalfx/pull/173)
* resource/time_chart: Fix some unclean plans caused by type conversions gone mad. [#174](https://github.com/terraform-providers/terraform-provider-signalfx/pull/174)


## 4.18.2 (March 13, 2020)

BUGFIXES:
* resource/aws_integration: Corrected bad service name for `AWS/VPN`.

## 4.18.1 (March 11, 2020)

IMPROVEMENTS:

* resource/aws_integration: Fixed a problem that caused some services in AWS integrations to not work. [#167](https://github.com/terraform-providers/terraform-provider-signalfx/pull/167)
* resource/aws_integration: Using `namespace_sync_rule` without filters no longer causes an unclean plan. [#170](https://github.com/terraform-providers/terraform-provider-signalfx/pull/170)
* resource/detector: Unsetting the `max_delay` or setting it to `0` should now correctly reset on the max delay on `apply` rather than unhelpfully doing nothing and leaving an unclean plan. #[171](https://github.com/terraform-providers/terraform-provider-signalfx/pull/171)


## 4.18.0 (March 04, 2020)

IMPROVEMENTS:

* resource/detector: Various documentation fixes, thanks [xp-1000](https://github.com/xp-1000)! [#166](https://github.com/terraform-providers/terraform-provider-signalfx/pull/166)

## 4.17.0 (March 03, 2020)

IMPROVEMENTS:

* resource/aws_integration: Added various new AWS services for validation. [#163](https://github.com/terraform-providers/terraform-provider-signalfx/pull/163)

## 4.16.0 (March 02, 2020)

IMPROVEMENTS:

* resource/aws_integration: The fields in `namespace_sync_rule` and `custom_namespace_sync_rule` are now optional except for `namespace`. [#161](https://github.com/terraform-providers/terraform-provider-signalfx/pull/161)

## 4.15.0 (February 26, 2020)

FEATURES:

* Added `signalfx_webhook_integration` resource. [#158](https://github.com/terraform-providers/terraform-provider-signalfx/pull/158)

## 4.14.0 (February 25, 2020)

BUG FIXES:

* resource/data_link: Use `property_name` when set without a `property_value`.

IMPROVEMENTS:

* Remove some interpolation-only expressions, which are now deprecated. [#152](https://github.com/terraform-providers/terraform-provider-signalfx/issues/152)
* resource/data_link: Add `"EpochSeconds"` as a value for `time_format`. [#156](https://github.com/terraform-providers/terraform-provider-signalfx/pull/156)

## 4.13.0 (February 13, 2020)

IMPROVEMENTS:

* resource/signalfx_aws_integration: Added support for `use_get_metric_data_method`.

## 4.12.2 (January 29, 2020)

IMPROVEMENTS:

## 4.12.1 (January 29, 2020)

* resource/time_chart: Fix accidental overzealous validation of `on_chart_legend_dimension` from last release. Sorry! [#145](https://github.com/terraform-providers/terraform-provider-signalfx/pull/145)

IMPROVEMENTS:

* resource/time_chart: Added validation for `on_chart_legend_dimension` to prevent unclean plans. [#143](https://github.com/terraform-providers/terraform-provider-signalfx/pull/143)

## 4.12.0 (January 27, 2020)

FEATURES:

* resource/detector: Added `viz_options` field and its constituents: `label`, `display_name`, `color`, `value_unit`, `value_prefix` and `value_suffix`.

## 4.11.1 (January 21, 2020)

IMPROVEMENTS:

* resource/aws_external_integration: Added field `signalfx_aws_account`, updated documentation. [#140](https://github.com/terraform-providers/terraform-provider-signalfx/pull/140)
* resource/heatmap_chart: Began validating `unit_prefix`. [#139](https://github.com/terraform-providers/terraform-provider-signalfx/pull/139)
* resource/list_chart: Added `time_range`, `start_time` and `end_time`. [#137](https://github.com/terraform-providers/terraform-provider-signalfx/pull/137)
* resource/list_chart: Began validating `color_by`. [#138](https://github.com/terraform-providers/terraform-provider-signalfx/pull/138)
* resource/list_chart: Began validating `unit_prefix`. [#139](https://github.com/terraform-providers/terraform-provider-signalfx/pull/139)
* resource/single_value_chart: Began validating `color_by`. [#136](https://github.com/terraform-providers/terraform-provider-signalfx/pull/136)
* resource/single_value_chart: Began validating `unit_prefix`. [#139](https://github.com/terraform-providers/terraform-provider-signalfx/pull/139)
* resource/time_chart: Began validating `color_by`. [#138](https://github.com/terraform-providers/terraform-provider-signalfx/pull/138)
* resource/time_chart: Began validating `unit_prefix`. [#139](https://github.com/terraform-providers/terraform-provider-signalfx/pull/139)

BUG FIXES:

* docs: Fix bad example of poll rate for GCP integration.
* docs: Document description field of Detector. [#134](https://github.com/terraform-providers/terraform-provider-signalfx/pull/134), thanks [@shwin](https://github.com/shwin)

## 4.11.0 (December 19, 2019)

FEATURES:

* provider: Added support for [Data Links](https://docs.signalfx.com/en/latest/managing/data-links.html) via `signalfx_data_link`. [#125](https://github.com/terraform-providers/terraform-provider-signalfx/pull/125)

IMPROVEMENTS:

* Fixed some little doc tings. [#131](https://github.com/terraform-providers/terraform-provider-signalfx/pull/131)

BUG FIXES:

* resource/detector: Webhook notifications are now validated as required a credentialId or a URL and secret. [#129](https://github.com/terraform-providers/terraform-provider-signalfx/pull/129)

## 4.10.3 (December 09, 2019)

BUG FIXES:

* resource/org_token: Fixed a problem where tokens with URL-unsafe names (those including slashes, etc) were not being URL encoded. [#123](https://github.com/terraform-providers/terraform-provider-signalfx/pull/123)

## 4.10.2 (December 04, 2019)

BUG FIXES:

* resource/dashboard: Fixed a crash for dashboards that were missing an "event signal" section. [#120](https://github.com/terraform-providers/terraform-provider-signalfx/pull/120)

## 4.10.1 (November 14, 2019)

BUG FIXES:

* resource/azure_integration: Fixed a bug where subscription IDs were incorrectly validated. [#113](https://github.com/terraform-providers/terraform-provider-signalfx/pull/113)

## 4.10.0 (November 07, 2019)

FEATURES:

* provider: Added `signalfx_alert_muting_rule` resource for managing alert muting rules. [#110](https://github.com/terraform-providers/terraform-provider-signalfx/pull/110)
* resource/dashboard: Added `authorized_writer_teams` and `authorized_writer_users` [#109](https://github.com/terraform-providers/terraform-provider-signalfx/pull/109)
* resource/dashboard_group: Added `authorized_writer_teams` and `authorized_writer_users` [#109](https://github.com/terraform-providers/terraform-provider-signalfx/pull/109)
* resource/detector: Added `authorized_writer_teams` and `authorized_writer_users` [#109](https://github.com/terraform-providers/terraform-provider-signalfx/pull/109)

## 4.9.2 (October 31, 2019)

FEATURES:

provider: Added support for Jira integrations via `signalfx_jira_integration`. [#106](https://github.com/terraform-providers/terraform-provider-signalfx/pull/106)
resource/detector: Added support for Jira notifications [#106](https://github.com/terraform-providers/terraform-provider-signalfx/pull/106)

BUG FIXES:

* resource/team: Documented the `members` property, which was unhelpfully undocumented previously.

## 4.9.1 (October 16, 2019)

BUG FIXES:

* resource/dashboard: Corrected validation of chart widths, allowing 12. Thanks [@ImDevinC](https://github.com/ImDevinC) [#100](https://github.com/terraform-providers/terraform-provider-signalfx/pull/100)

IMPROVEMENTS:

* resource/dashboard: Multiple instances of `column` and `grid` can now be used. [#102](https://github.com/terraform-providers/terraform-provider-signalfx/pull/102)

## 4.9.0 (October 10, 2019)

FEATURES:

* provider: Added `signalfx_team` resource. [#5](https://github.com/terraform-providers/terraform-provider-signalfx/pull/5)

BUG FIXES:

* resource/heatmap_chart: Now check that one of `color_range` or `color_scale` is set and emit an error if not. [#96](https://github.com/terraform-providers/terraform-provider-signalfx/pull/96)

IMPROVEMENTS

* resource/list_chart: An error is now emitted if `color_scale` is used without a `color_by = "Scale"`. [#99](https://github.com/terraform-providers/terraform-provider-signalfx/pull/99)

## 4.8.3 (September 27, 2019)

IMPROVEMENTS:

* provider: Updated to terraform-plugin-sdk [#93](https://github.com/terraform-providers/terraform-provider-signalfx/pull/93)
* provider: Updated to new signalfx-go dep, prevent possible crashes from JSON changes.

## 4.8.2 (September 26, 2019)

BUG FIXES:

* resource/time_chart: Fix crash when importing some charts with only a left axis. [#92](https://github.com/terraform-providers/terraform-provider-signalfx/pull/92)

## 4.8.1 (September 23, 2019)

FEATURES:

* resource/heatmap_chart: Now supports the `color_scale` option. [#89](https://github.com/terraform-providers/terraform-provider-signalfx/pull/89)

BUG FIXES:

* resource/heatmap_chart: No longer allows setting multiple `color_range` options. [#89](https://github.com/terraform-providers/terraform-provider-signalfx/pull/89)
* resource/heatmap_chart: Many integer fields now verify that the value is >= 0 [#89](https://github.com/terraform-providers/terraform-provider-signalfx/pull/89)
* resource/heatmap_chart: The `color_range.color` property was confusingly allowing both hex and non-hex colors. This has been standardized to hex colors. This may generate errors and ask you to change your colors if you used the old form. [#89](https://github.com/terraform-providers/terraform-provider-signalfx/pull/89)
* resource/detector: Improved guards against null values from detectors that might cause a crash and added more property validation in the schema. Thanks to [@joshuaspence](https://github.com/joshuaspence) for flagging. [#91](https://github.com/terraform-providers/terraform-provider-signalfx/pull/91)

## 4.8.0 (September 19, 2019)

FEATURES:

* provider: Added `signalfx_aws_external_integration` and `signalfx_aws_token_integration` resources to improve AWS management.

BUG FIXES:

* resource/dashboard: Use of `column` was causing unclean plans. [#85](https://github.com/terraform-providers/terraform-provider-signalfx/pull/85)
* resource/detector: Add default for `time_range`, which was being set by the API and causing unclean plans. [#83](https://github.com/terraform-providers/terraform-provider-signalfx/pull/83)
* resource/detector: Correct cast of start, end, and range times to `int64`. [#87](https://github.com/terraform-providers/terraform-provider-signalfx/pull/87)

BACKWARDS INCOMPATIBILITIES:

* resource/aws_integration: To allow fully in-Terraform management of AWS integrations, added new resources `signalfx_aws_external_integration` and `signalfx_aws_token_integration` to be used in conjunction with `signalfx_aws_integration`. This changes some of the fields to be computed. These changes are documented in the documentation for the aforementioned resources.

## 4.7.0 (September 17, 2019)

FEATURES:

provider: Errors related to 4xx statuses when manipulating integrations now hint that you might need to use an admin token. Also added notes to the docs for same. [#70](https://github.com/terraform-providers/terraform-provider-signalfx/pull/70)
provider: Added VictorOps integration resource. [#79](https://github.com/terraform-providers/terraform-provider-signalfx/pull/79)

BUG FIXES:

* provider: Documentation page titles now reflect the actual resource name. [#79](https://github.com/terraform-providers/terraform-provider-signalfx/pull/79)
* resource/dashboard: Dashboard variables with no default value no longer cause unclean plans. [#68](https://github.com/terraform-providers/terraform-provider-signalfx/pull/68)
* resource/time_chart: Corrected an error in the document that made `event_options` look to be nested under `viz_options`. It is not!
* resource/time_chart: Corrected documentation for `legend_options_fields.property`'s "special" values `metric` and `plot_label`. (Also for resource/list_chart). [#77](https://github.com/terraform-providers/terraform-provider-signalfx/pull/77)
* resource/heatmap_chart: Correctly validate `color_range` and adjust docs to demonstrate proper input of hex colors. [#76](https://github.com/terraform-providers/terraform-provider-signalfx/pull/76)

IMPROVEMENTS:

* provider: Upgraded to Terraform library v0.12.8

## 4.6.3 (August 21, 2019)

BUG FIXES:

* resource/time_chart: Corrected an crash when using `event_options`. [#63](https://github.com/terraform-providers/terraform-provider-signalfx/pull/64)

## 4.6.2 (August 20, 2019)

FEATURES:

* resource/time_chart: Added `event_options` to support cutomization of events

## 4.6.1 (August 16, 2019)

BUG FIXES:

* resource/detector: Fixed a bug in unmarshaling Opsgenie notifications. [#60]https://github.com/terraform-providers/terraform-provider-signalfx/pull/60

## 4.6.0 (August 15, 2019)

FEATURES:

* Added `resource_opsgenie_integration`. [#54](https://github.com/terraform-providers/terraform-provider-signalfx/pull/54)

BUG FIXES:

* provider: Fixed the documentation sidebar which had a number of incorrect integration resource names. [#53](https://github.com/terraform-providers/terraform-provider-signalfx/pull/53)
* resource/time_chart: Fix incorrect documentation around use of `time_range`. [#56](https://github.com/terraform-providers/terraform-provider-signalfx/pull/56)
* resource/time_chart: Correct unclean plans when using `on_chart_legend_dimension`. [#58](https://github.com/terraform-providers/terraform-provider-signalfx/pull/58)

IMPROVEMENTS:

* resource/pagerduty_integration: Added `Exists` functionality, enabled acceptance tests, and use the new `*GCPIntegration` methods from signalfx-go. [#51](https://github.com/terraform-providers/terraform-provider-signalfx/pull/51)
* resource/gcp_integration: Added `Exists` functionality, enabled acceptance tests, and use the new `*GCPIntegration` methods from signalfx-go. [#50](https://github.com/terraform-providers/terraform-provider-signalfx/pull/50)
* resource/slack_integration: Added `Exists` functionality, enabled acceptance tests, and use the new `*GCPIntegration` methods from signalfx-go. [#52](https://github.com/terraform-providers/terraform-provider-signalfx/pull/52)

BACKWARDS INCOMPATIBILITIES:

* resource/integration: This resource was removed, as it had been deprecated for a while. [#52](https://github.com/terraform-providers/terraform-provider-signalfx/pull/52)

## 4.5.0 (August 09, 2019)

FEATURES:

* provider: Added support for Organization Tokens with `signalfx_org_token`. [#45](https://github.com/terraform-providers/terraform-provider-signalfx/pull/45)

IMPROVEMENTS:

* provider: Bumped Terraform dependency to v0.12.6 [#47](https://github.com/terraform-providers/terraform-provider-signalfx/pull/47)
* resource/gcp_integration: Improve the GCP documentation example. Thanks [a-staebler](https://github.com/a-staebler) [#41](https://github.com/terraform-providers/terraform-provider-signalfx/pull/41)
* resource/detector: Notifications are now validated to prevent crashes and problems. [#46](https://github.com/terraform-providers/terraform-provider-signalfx/pull/46)
* resource/detector: Fixed a bug in Webhook notification specifications, it was missing a `credentialId`.
* resource/detector: Corrected documentation that disagreed on whether to include `#` in Slack channel names. In a word: don't.
* resource/detector: Improve type checking and reliability of notification strings. [#48](https://github.com/terraform-providers/terraform-provider-signalfx/pull/48)

## 4.4.0 (July 30, 2019)

FEATURES:

* provider: Added support for Azure integrations [#34](https://github.com/terraform-providers/terraform-provider-signalfx/pull/34)

BUG FIXES:

* provider: Resources that had gone missing were not recreated, but instead threw errors. Those resources will now be recreated. [#38](https://github.com/terraform-providers/terraform-provider-signalfx/pull/38)
* resource/time_chart: The axis' low watermark, if unset, could get "stuck" and insist on needing to change the remote chart, leaving an unclean `apply`. This has been fixed by correcting the default value, which was set incorrectly. [#35](https://github.com/terraform-providers/terraform-provider-signalfx/issues/35)

IMPROVEMENTS:

* provider: Added AWS resource link to documentation sidebar. [#37](https://github.com/terraform-providers/terraform-provider-signalfx/pull/37)
* resources/detector: Improved documentation for OpsGenie notifications. Thanks [austburn](https://github.com/austburn)! Thanks [#36](https://github.com/terraform-providers/terraform-provider-signalfx/pull/36).
* resources/time_chart: `axis_left` and `axis_right` are now limited to single uses. This was always the case, but it's now enforced in the schema to prevent blissful ignorance.

## 4.3.0 (July 24, 2019)

FEATURES:

* provider: Added support for AWS integrations [#32](https://github.com/terraform-providers/terraform-provider-signalfx/pull/32)

BUG FIXES:

* resource/pagerduty_integration: Fixed incorrect documentation. [#32](https://github.com/terraform-providers/terraform-provider-signalfx/pull/32)

IMPROVEMENTS:

* resources/detector: Improved documentation around multiple notifications in a single rule. [#30](https://github.com/terraform-providers/terraform-provider-signalfx/issues/30)

## 4.2.0 (July 19, 2019)

FEATURES:

* resource/time_chart: Added support for `viz_options.display_name` [#13](https://github.com/terraform-providers/terraform-provider-signalfx/issues/13)
* resource/list_chart: Added support for `viz_options.display_name` [#13](https://github.com/terraform-providers/terraform-provider-signalfx/issues/13)
* resource/single_value_chart: Added support for `viz_options.display_name` [#13](https://github.com/terraform-providers/terraform-provider-signalfx/issues/13)

BUG FIXES:

* provider: Fixed a number of fields that were not correctly imported. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)
* resource/detector: Fixed incorrect documentation for Slack notifications. Thanks [gpetrousov](https://github.com/gpetrousov). [#25](https://github.com/terraform-providers/terraform-provider-signalfx/issues/25)
* resource/detector: Fixed invalid field for OpsGenie notifications. [#16](https://github.com/terraform-providers/terraform-provider-signalfx/issues/16)
* resource/list_chart: Fixed an issue where `viz_options` was not being honored. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)
* resource/single_value_chart: Fixed an issue where `viz_options` was not being honored. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)
* resource/time_chart: Fixed crash where specifying `histogram_options.color_theme` would cause a crash. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)
* resource/time_chart: `show_data_markers` no longer defaults to `false` because it is often omitted from API responses. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)

IMPROVEMENTS

* provider - Corrected places where resources were double-setting their URLs. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)
* provider - Added import tests to all resources. [#27](https://github.com/terraform-providers/terraform-provider-signalfx/pull/27)

## 4.1.0 (July 17, 2019)

FEATURES:

* resource/dashboard_group: Add support for [Mirrored Dashboards](https://docs.signalfx.com/en/latest/dashboards/dashboard-mirrors.html) [#4](https://github.com/terraform-providers/terraform-provider-signalfx/issues/4)

BUG FIXES:

* provider: Bump [signalfx-go](https://github.com/signalfx/signalfx-go) dependency to [v1.2.0](https://github.com/signalfx/signalfx-go/blob/master/CHANGELOG.md#120-2019-07-16) which fixes a regression in creating "empty" dashboards with any new dashboard group. [#14](https://github.com/terraform-providers/terraform-provider-signalfx/issues/14)
* provider: Correct a number of fields that defaulted to 0, resulting unintentional "defaults". Should improve unclean plans.
* resource/dashboard: Fix a crash when using `grid` with a new dashboard. [#20](https://github.com/terraform-providers/terraform-provider-signalfx/issues/20)

IMPROVEMENTS:

* provider - Resources that used `time_range` and still have strings in their state will now be upgraded instead of generating an error.

## 4.0.0 (July 08, 2019)

NOTES:

* provider: This provider is now targeting Terraform 0.12, users can find support for 0.11 and earlier in the branch `tf-11-compat`.
* provider: After upgrading users may find minor changes to otherwise clean state. These are likely the result of new default values for many properties that previously lacked them.
* provider: Now emits useful messages into debug logs in case they are needed. (They are for the author!)
* provider: This provider previously ignored the response body of API calls and wrote the state file without considering the document that was returned. It is considered idiomatic in Terraform to either read the response or issue a follow-up `GET` request to hydrate the state using the API's version of the document. This being absent allowed a number of oddities in this provider which have been fixed.
* resource/signalfx_time_chart: Property `legend_fields_to_hide` has been deprecated. Please use `legend_options_fields` instead.
* resource/signalfx_list_chart: Property `legend_fields_to_hide` has been deprecated. Please use `legend_options_fields` instead.

FEATURES:

* provider: Added various utility methods for color name and index lookups
* resources/detector: Added support for BigPanda, Office365, ServiceNow, xMatters and VictorOps notification types [#49](https://github.com/signalfx/terraform-provider-signalfx/issues/49)
* resource/event_feed_chart: Add properties `time_range`, `start_time`, and `end_time`.
* resource/list_chart: now supports `legend_options_fields`.
* resource/list_chart: now support `color_scale` and it's sub-fields [#76](https://github.com/signalfx/terraform-provider-signalfx/pull/76)
* resource/time_chart: now supports `timezone`, thanks [zimingw](https://github.com/zimingw). [#60](https://github.com/signalfx/terraform-provider-signalfx/pull/60) and [#68](https://github.com/signalfx/terraform-provider-signalfx/pull/68)


BUG FIXES:

* provider: All resources lacked property acceptance tests that verified proper state function. These tests have now been added.
* provider: Many resource properties now include default values.
* provider: Documentation for `value_unit` has been improved with valid values. [#53](https://github.com/signalfx/terraform-provider-signalfx/issues/53)
* resource/detector: `tags` has been removed
* resource/event_feed_chart: `viz_options` has been removed
* resource/list_chart: Improved docs and examples for `legend_options_fields`. [#65](https://github.com/signalfx/terraform-provider-signalfx/pull/65)
* resource/list_chart: Documented `viz_options` and it's sub-properties. [#57](https://github.com/signalfx/terraform-provider-signalfx/issues/57)
* resource/time_chart: Improved docs and examples for `legend_options_fields`. [#65](https://github.com/signalfx/terraform-provider-signalfx/pull/65)

IMPROVEMENTS:

* provider: Most resources now implement an `Exists` function which verifies the asset exists and adjusts the plan accordingly. [#75](https://github.com/signalfx/terraform-provider-signalfx/pull/75)
* resource/list_chart: Parameters for `sort_by` have had documentation improved. [#64](https://github.com/signalfx/terraform-provider-signalfx/pull/64)
* resource/time_chart: `axes_precision` property now has documentation. [#55](https://github.com/signalfx/terraform-provider-signalfx/issues/55)
* resource/dashboard: Corrected name of `filter.negated` which was incorrectly documented as `not`. [#52](https://github.com/signalfx/terraform-provider-signalfx/issues/52)
* resource/detector: Added examples for `Team` and `TeamEmail` notifications. [#50](https://github.com/signalfx/terraform-provider-signalfx/issues/50)

BACKWARDS INCOMPATIBILITIES:

* provider: There is no longer a `synced` attribute of all non-integration resources. This computed property reflected whether or not the `last_updated` property had changed on the API-side of SignalFx. It acted as a signal for the operator that the remote resource had changed without Terraform's knowledge. While useful in some situations this behavior is non-idiomatic in Terraform. This has the side effect of cleaning up plan/apply output for many users who didn't know what `synced` meant.
* provider: The attribute `time_range` of various resources has changed from `String` to `Int`. Values like `1h` must now be expressed in seconds. For example `1h` should become `3600` as that's how many seconds are in an hour.
* provider: The `last_updated` attribute was removed from all non-integration resources, as it was no longer needed when `sync` was removed.
* resource/signalfx_dashboard: The property `tags` has been removed from to prevent race conditions.
* resource/signalfx_dashboard: You may no longer mix `grid`, `column` and `chart` in a dashboard.
* resource/signalfx_dashboard: If you use `grid` or `column` you can only use them one time.
* resource/signalfx_dashboard: `grid.start_row` has been removed
* resource/signalfx_dashboard: `grid.start_column` has been removed
* resource/signalfx_dashboard: `column.start_row` has been removed
* resource/signalfx_detector: The property `tags` has been removed from to prevent race conditions.
* resource/signalfx_event_feed_chart: removed the `viz_options` block and it's constituent `label` and `color` since they didn't do anything.
* resource/signalfx_heatmap_chart: no longer tries to do anything with `color_by` of `"Scale"` as the code that was there didn't send valid data.

## 3.3.0 (2019-06-28)

## Added

* Added docs for Slack integration.
* Added acceptance tests for Integration and Detector.
* [New resource](https://github.com/signalfx/terraform-provider-signalfx/pull/35) `signalfx_event_feed_chart` for [Event Feed charts](https://docs.signalfx.com/en/latest/dashboards/dashboard-add-info.html#adding-an-event-feed-chart-to-a-dashboard).
* [New resources](https://github.com/signalfx/terraform-provider-signalfx/pull/34) `resource_pagerduty_integration` and `resource_gcp_integration` which completes the trifecta needed to get rid of `resource_integration` in the future.
* [Added 'refresh_interval' property to Heatmap](https://github.com/signalfx/terraform-provider-signalfx/pull/45). Thanks to [clayembry](https://github.com/clayembry) for flagging.

## Fixed

* [Adjusted](https://github.com/signalfx/terraform-provider-signalfx/pull/28) confusing docs for dashboard event overlays. Thanks to [detouched](https://github.com/detouched) for flagging!

## Changed

* Added Go module vendor directory per [HashiCorp guidelines](https://github.com/signalfx/terraform-provider-signalfx/issues/37)

## 3.2.0 (2019-05-24)

## Added
* Start of [acceptance tests](https://github.com/signalfx/terraform-provider-signalfx/pull/24) (dashboards, charts, dashboard groups)
* Use of [signalfx-go](https://github.com/signalfx/signalfx-go) in acceptance tests, with plans to use it for all API calls in the future.
* New property `legend_options_fields` for [Time Charts](https://github.com/signalfx/terraform-provider-signalfx/blob/master/docs/resources/time_chart.md#argument-reference) and [List Charts](https://github.com/signalfx/terraform-provider-signalfx/blob/master/docs/resources/list_chart.md#argument-reference). This allows ordering and toggling of individual properties in the data table.

### New Integration Style, Preview
SignalFx's Integration API uses a single endpoint for all calls, but varies the JSON document that is submitted. As such, this provider follows the same convention, using `signalfx_integration` as a resource with a bunch of mixed keys.

In this release we've [added a new resource for PagerDuty integrations](https://github.com/signalfx/terraform-provider-signalfx/pull/21) called `signalfx_pagerduty_integration`. It matches the existing use of `signalfx_integration` with a `type = "PagerDuty"`.

It is expected that this form of specific integrations will replace the generic one. This is a boon for maintenance and more explicit for users.

Please open issues if you have comments, and feel free to use this resource. A future release will deprecate and remove `signalfx_integration` if all goes well.

## Fixed

* Fixed some busted links in documentation.
* Fixed [bug](https://github.com/signalfx/terraform-provider-signalfx/issues/15) in docs for Single Value Charts and appropriate values for `color_by`. Thanks [MovieStoreGuy](https://github.com/MovieStoreGuy) for reporting and [draquila](https://github.com/draquila) for suggesting the fix.
* Creating a Dashboard Group [no longer implicitly creates an empty dashboard of the same name](https://github.com/signalfx/terraform-provider-signalfx/pull/23) as a member of the group. Note: This will *not* remove any dashboards previously created that way, nor will it prevent you from creating a dashboard group with nothing in it. That's on you! Thanks to [MovieStoreGuy](https://github.com/MovieStoreGuy) for flagging this.
* Charts no longer [silently fail](https://github.com/signalfx/terraform-provider-signalfx/pull/25) to create on errors. Thanks [djmason](https://github.com/djmason)!
* Fixed a bug in the example for single value charts.

## Changed
* Bumped terraform dependency version
* Adjusted some tests to deal with having `SFX_AUTH_TOKEN` set when running acceptance tests.

## 3.1.0 (2019-05-21)

* Any use of a resource's `resource_url` should be replaced with `url`, most likely as an output value. See the Removed section below for more.

## Added

* [Support](https://github.com/signalfx/terraform-provider-signalfx/pull/16) for OpsGenie as a notifier for detectors, thanks [juliawong](https://github.com/juliawong).
* GCP integrations are [now supported](https://github.com/signalfx/terraform-provider-signalfx/pull/17) by the [`signalfx_integration` resource](https://github.com/signalfx/terraform-provider-signalfx/blob/master/docs/resources/integration.md). Thanks [seonaidm](https://github.com/signalfx/terraform-provider-signalfx/pull/17)!

### New Configuration Options

The following new options were added to the provider's configuration:

* `api_url` which allows users to customize the URL to which API requests will be sent. This allows users of other realms or those using proxies to use this provider. Note you probably want to change `custom_app_url` as well!
* `custom_app_url` which allows users to customize the app URL used for viewing resources. This is used by organizations using specific realms or those with a custom [SSO domain](https://docs.signalfx.com/en/latest/admin-guide/sso.html).

## Fixed

* Fixed some busted links in documentation.
* Fixed [bug](https://github.com/signalfx/terraform-provider-signalfx/issues/15) in docs for Single Value Charts and appropriate values for `color_by`. Thanks [MovieStoreGuy](https://github.com/MovieStoreGuy) for reporting and [draquila](https://github.com/draquila) for suggesting the fix.

## Removed

* The attribute `resource_url` has been removed from resources. This means that the provider will not output a URL after an `apply`, since the `url` resource is "computed" in Terraform parlance. You can, however, find the URL for any asset with `terraform show <asset name>`. For example, `terraform state show signalfx_dashboard.mydashboard1`.

## 3.0.0 (2019-03-18)

We're jumping to a 3.0.0 version number after forking from [Yelp's SignalForm](https://github.com/Yelp/terraform-provider-signalform/), incorporating [Stripe's fork](https://github.com/stripe/terraform-provider-signalform/), and renaming to `terraform-provider-signalfx`.

Thanks to Yelp and Stripe for their contributions!

## Added

* Added `axes_include_zero` option for time charts: [PR](https://github.com/stripe/terraform-provider-signalform/pull/1)
* Added `event_overlay` and `selected_event_overlay` support: [PR](https://github.com/stripe/terraform-provider-signalform/pull/5)
* Added `show_event_lines` and `disable_sampling` to detectors: [PR](https://github.com/stripe/terraform-provider-signalform/pull/9)
* Added `type` to dashboard event overlays to allow the usage of detector events in addition to the (default) custom events: [PR](https://github.com/stripe/terraform-provider-signalform/pull/9)
* Added `secondary_visualization` for list charts and single value charts: [PR](https://github.com/stripe/terraform-provider-signalform/pull/10)
* Added `histogram_options` and it's descendant `color_theme` to time charts: [PR](https://github.com/stripe/terraform-provider-signalform/pull/14)

## Updated

* Now uses Go 1.12 and module support, dropping use of Glide.
* PagerDuty: Opt out of sending validation messages and add necessary friends: [PR](https://github.com/stripe/terraform-provider-signalform/pull/3)

## Bugfixes

* Fixed some bugs in docs for single-value charts: [PR](https://github.com/stripe/terraform-provider-signalform/pull/6)
* Avoid panics from type assertions on nils: [PR](https://github.com/stripe/terraform-provider-signalform/pull/9)
* Fix color scale handling in single value charts: [PR](https://github.com/stripe/terraform-provider-signalform/pull/9)

## Removed

* This provider no longer attempts to sanitize SignalFlow program text, as doing so was causing problems with indentation.

# Old Changelog

terraform-provider-signalform (2.8.0) trusty; urgency=low

  * added tags to charts and dashboards

 -- Wendy Vivar <wendyv@yelp.com>  Thu, 31 Jan 2019 14:09:12 -0800

terraform-provider-signalform (2.7.1) trusty; urgency=medium

  * use correct colors in colorscale

 -- Timothy Mower <tmower@yelp.com>  Fri, 06 Jul 2018 03:36:47 -0700

terraform-provider-signalform (2.7.0) trusty; urgency=low

  * Added teams to dashboard groups.
  * Added support for valueUnit, publishLabelOptions and other options
    to charts.
  * Allow detector rules to notify teams.

 -- Rahul Ravindran <rahulrav@yelp.com>  Thu, 05 Apr 2018 16:30:29 -0700

terraform-provider-signalform (2.6.0) trusty; urgency=low

  * Added netrc support

 -- Francesco Di Chiara <fdc@yelp.com>  Mon, 12 Feb 2018 11:51:35 -0800

terraform-provider-signalform (2.5.1) trusty; urgency=low

  * Freeze lib ffi version

 -- Rahul Ravindran <rahulrav@yelp.com>  Wed, 07 Feb 2018 09:46:35 -0800

terraform-provider-signalform (2.5.0) trusty; urgency=low

  * Support creation of runbook_url and tip in detector model.

 -- Rahul Ravindran <rahulrav@yelp.com>  Tue, 06 Feb 2018 10:47:00 -0800

terraform-provider-signalform (2.4.0) trusty; urgency=low

  * Change way to add scale colors to heatmaps. Add slack notif integration

 -- Francesco Di Chiara <fdc@yelp.com>  Wed, 31 Jan 2018 09:52:36 -0800

terraform-provider-signalform (2.3.0) trusty; urgency=low

  * Support new parameters in detectors.

 -- Rahul Ravindran <rahulrav@yelp.com>  Fri, 15 Dec 2017 11:16:03 -0800

terraform-provider-signalform (2.2.9) trusty; urgency=low

  * Building against terraform 0.10

 -- Sargurunathan Mohan <sargurum@yelp.com>  Thu, 02 Nov 2017 02:35:26 -0700

terraform-provider-signalform (2.2.8) trusty; urgency=low

  * Fix threshold type from int to float.

 -- Francesco Di Chiara <fdc@yelp.com>  Wed, 01 Nov 2017 07:41:01 -0700

terraform-provider-signalform (2.2.7) trusty; urgency=low

  * Initial public release.

 -- Antonio Verardi <antonio@yelp.com>  Mon, 09 Oct 2017 08:13:35 -0700
