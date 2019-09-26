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

* provider: emits useful messages into debug logs in case they are needed. (They are for the author!)
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
