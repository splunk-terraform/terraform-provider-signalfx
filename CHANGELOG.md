# 3.3.0 (Unreleased)

## Added

* Added docs for Slack integration.
* Added acceptance tests for Integration and Detector.
* [New resource](https://github.com/signalfx/terraform-provider-signalfx/pull/35) `signalfx_event_feed_chart` for [Event Feed charts](https://docs.signalfx.com/en/latest/dashboards/dashboard-add-info.html#adding-an-event-feed-chart-to-a-dashboard).
* [New resources](https://github.com/signalfx/terraform-provider-signalfx/pull/34) `resource_pagerduty_integration` and `resource_gcp_integration` which completes the trifecta needed to get rid of `resource_integration` in the future.
* [Added 'refresh_interval' property to Heatmap](https://github.com/signalfx/terraform-provider-signalfx/pull/45). Thanks to [clayembry](https://github.com/clayembry) for flagging.

## Fixed

* [Adjusted](https://github.com/signalfx/terraform-provider-signalfx/pull/28) confusing docs for dashboard event overlays. Thanks to [detouched](https://github.com/detouched) for flagging!

## Removed

## Changed

* Added Go module vendor directory per [HashiCorp guidelines](https://github.com/signalfx/terraform-provider-signalfx/issues/37)

# 3.2.0, 2019-05-24

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

# 3.1.0, 2019-05-21

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

# 3.0.0, 2019-03-18

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
