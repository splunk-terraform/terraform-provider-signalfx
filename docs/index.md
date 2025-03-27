---
page_title: "Provider: SignalFx"
description: |-
  Use the Splunk Observability Cloud provider, formerly known as SignalFx Terraform provider, to interact with the resources supported by Splunk Observability Cloud. Configure the provider with the proper credentials before using it.
---

# Splunk Observability Cloud provider

The [Splunk Observability Cloud](https://www.splunk.com/en_us/products/observability.html) provider, formerly known as SignalFx Terraform provider, lets you interact with the resources supported by Splunk Observability Cloud. You must configure the provider with credentials before using it.

Use the navigation to learn more about the available resources.

# Learn about Splunk Observability Cloud

To learn more about Splunk Observability Cloud and its features, see [the official documentation](https://docs.splunk.com/observability/en/).

You can use the SignalFlow programming language to create charts and detectors using `program_text`. For more information about SignalFlow, see the [Splunk developer documentation](https://dev.splunk.com/observability/docs/signalflow/).

# Authentication

When authenticating to the Splunk Observability Cloud API you can use:

1. An Org token.
2. A Session token.
3. A Service account.

See [Authenticate API Requests](https://dev.splunk.com/observability/docs/apibasics/authentication_basics/) in the Splunk developer documentation.

Session tokens are short-lived and provide administrative permissions to edit integrations. They expire relatively quickly, but let you manipulate some sensitive resources. Resources that require session tokens are flagged in their documentation.

A Service account is term used when a user is created within organization that can login via Username and Password, this allows for a *Session Token* to be created by the terraform provider and then used throughout the application.

ℹ️ **NOTE** Separate the less sensitive resources, such as dashboards, from the more sensitive ones, such as integrations, to avoid having to change tokens.

## Example

The following example shows how to configure the Splunk Observability Cloud provider for Terraform:

```terraform
# Configure the Splunk Observability Cloud provider
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

```terraform
# An example configuration with a service account.
provider "signalfx" {
  email           = "service.account@example"
  password        = "${var.service_account_password}"
  organization_id = "${var.service_account_org_id}"
  # If your organization uses a different realm
  # api_url = "https://api.<realm>.signalfx.com"
  # If your organization uses a custom URL
  # custom_app_url = "https://myorg.signalfx.com"
}
```

# Feature Previews

To allow for more experimental features to be added into the provider, a feature can be added behind a preview gate that defaults to being off and requires a user to opt into the change. Once a feature has been added into the provider, in can be set to globally available which will default to the feature being on by default.

There is an opportunity for the user to opt out of a globally available feature if an issue is experienced. If that is the case, please raise a support case with the provider configuration and any error messages.

The feature preview can be enabled by the following example:

```terraform
provider "signalfx" {
  # Other configured values
  feature_preview = {
    "feature-01": true,  // True means that the feature is enabled
    "feature-02": false, // False means that the feature is explicitly disabled
  }
}
```

ℹ️ **NOTE** Preview features are a subject to change and/or removal in a future version of the provider.

## Arguments

The provider supports the following arguments:

* `auth_token` - (Required) The auth token for [authentication](https://developers.signalfx.com/basics/authentication.html). You can also set it using the `SFX_AUTH_TOKEN` environment variable.
* `api_url` - (Optional) The API URL to use for communicating with Splunk Observability Cloud. This is helpful for organizations that need to set their realm or use a proxy. You can also set it using the `SFX_API_URL` environment variable.
* `custom_app_url` - (Optional) The application URL that users might use to interact with assets in the browser. Used by organizations on specific realms or with a custom [SSO domain](https://docs.splunk.com/observability/en/admin/authentication/SSO/sso-about.html). You can also set it using the `SFX_CUSTOM_APP_URL` environment variable.
* `timeout_seconds` - (Optional) The total timeout duration to wait when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `120`.
* `retry_max_attempts` - (Optional) The number of retry attempts when making HTTP API calls to Splunk Observability Cloud. Defaults to `4`.
* `retry_wait_min_seconds` - (Optional) The minimum wait time between retry attempts when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `1`.
* `retry_wait_max_seconds` - (Optional) The maximum wait time between retry attempts when making HTTP API calls to Splunk Observability Cloud, in seconds. Defaults to `30`.
* `email` - (Optional) The provided email address is used to generate a *Session Token* that is then used for all API interactions. Requires email address to be configured with a password, and not via SSO.
* `password` - (Optional) The password is used to authenticate the email provided to generate a *Session Token*. Requires email address to be configured with a password, and not via SSO.
* `organization_id` - (Optional) The organisation id is used to select which organization if the user provided belongs to multiple.
* `feature_preview` - (Optional) A map structure that allows users to enable experimental features before they are ready to be globally available.
