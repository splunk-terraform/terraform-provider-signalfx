---
layout: "signalfx"
page_title: "Provider: SignalFx"
sidebar_current: "docs-signalfx-index"
description: |-
  Use the Splunk Observability Cloud provider, formerly known as SignalFx provider, to interact with the resources supported by Splunk Observability Cloud. Configure the provider with the proper credentials before using it.
---

# Splunk Observability Cloud provider

The [Splunk Observability Cloud](https://www.splunk.com/en_us/products/observability.html) provider lets you interact with 
the resources supported by Splunk Observability Cloud. You must configure the provider with credentials before using it.

Use the navigation to learn more about the available resources.

# Learn about Splunk Observability Cloud

To learn more about Splunk Observability Cloud and its features, see [the official documentation](https://docs.splunk.com/observability/en/).

You can use the SignalFlow programming language to create charts and detectors using `program_text`. For more information about SignalFlow, see the [Splunk developer documentation](https://dev.splunk.com/observability/docs/signalflow/).

# Authentication

When authenticating to the Splunk Observability Cloud API you can use either an Org token or a 
Session token. See [Authenticate API Requests](https://dev.splunk.com/observability/docs/apibasics/authentication_basics/) in the Splunk developer documentation.

Session tokens are short-lived and provide administrative permissions to edit integrations. They expire relatively quickly, but let you manipulate some sensitive resources. Resources that require session tokens are flagged in their documentation.

~> **NOTE** Separate the less sensitive resources, such as dashboards, from the 
more sensitive ones, such as integrations, to avoid having to change tokens.

## Example

The following example shows how to configure the Splunk Observability Cloud provider for Terraform:

```hcl
# Configure the SignalFx provider
provider "signalfx" {
  auth_token = "${var.signalfx_auth_token}"
  # If your organization uses a different realm
  # api_url = "https://api.<realm>.signalfx.com"
  # If your organization uses a custom URL
  # custom_app_url = "https://myorg.signalfx.com"
}

# Create a new detector
resource "signalfx_detector" "default" {
  # ...
}

# Create a new dashboard
resource "signalfx_dashboard" "default" {
  # ...
}
```

## Arguments

The provider supports the following arguments:

* `auth_token` - (Required) The auth token for [authentication](https://developers.signalfx.com/basics/authentication.html). You can also set it using the `SFX_AUTH_TOKEN` environment variable.
* `api_url` - (Optional) The API URL to use for communicating with Splunk Observability Cloud. This is helpful for organizations that need to set their realm or use a proxy. You can also set it using the `SFX_API_URL` environment variable.
* `custom_app_url` - (Optional) The application URL that users might use to interact with assets in the browser. Used by organizations on specific realms or with a custom [SSO domain](https://docs.splunk.com/observability/en/admin/authentication/SSO/sso-about.html). You can also set it using the `SFX_CUSTOM_APP_URL` environment variable.
* `timeout_seconds` - (Optional) The total timeout duration to wait when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `120`.
* `retry_max_attempts` - (Optional) The number of retry attempts when making HTTP API calls to Splunk Observability Cloud. Defaults to `4`.
* `retry_wait_min_seconds` - (Optional) The minimum wait time between retry attempts when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `1`.
* `retry_wait_max_seconds` - (Optional) The maximum wait time between retry attempts when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `30`.
