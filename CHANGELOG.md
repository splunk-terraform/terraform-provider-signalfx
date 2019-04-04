# 3.1.0, in progress

* Any use of a resource's `resource_url` should be replaced with `url`, most likely as an output value. See the Removed section below for more.

## Added

### New Configuration Options

The following new options were added to the provider's configuration:

* `api_url` which allows users to customize the URL to which API requests will be sent. This allows users of other realms or those using proxies to use this provider. Note you probably want to change `custom_app_url` as well!
* `custom_app_url` which allows users to customize the app URL used for viewing resources. This is used by organizations using specific realms or those with a custom [SSO domain](https://docs.signalfx.com/en/latest/admin-guide/sso.html).

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
