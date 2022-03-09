---
layout: "signalfx"
page_title: "Provider: SignalFx"
sidebar_current: "docs-signalfx-index"
description: |-
  The SignalFx provider is used to interact with the resources supported by SignalFx. The provider needs to be configured with the proper credentials before it can be used.
---

# SignalFx provider

The [SignalFx](https://www.signalfx.com/) provider lets you interact with 
the resources supported by SignalFx. You must configure the provider with 
credentials before using it.

Use the navigation to learn more about the available resources.

# Learning about Terraform and SignalFx

Terraform is a powerful tool and ecosystem beyond. See [Introduction to Terraform](https://www.terraform.io/intro/index.html)
to understand the basics of working with Terraform. 

See [SignalFlow programming language](https://dev.splunk.com/observability/docs/signalflow/) to learn how to create 
SignalFx charts and detectors using `program_text`. You can click "View SignalFlow" when editing any chart in Splunk Observability Cloud to see the underlying code.

# Authentication

When authenticating to the SignalFx API you can use either an Org token or a 
Session token. See [Authenticate API Requests](https://dev.splunk.com/observability/docs/apibasics/authentication_basics/) for more
information.

Session tokens are short-lived and provide administrative permissions to edit integrations. These expire relatively quickly, but allow the manipulation of some more sensitive resources. Resources that require this are flagged in their documentation.

~> **NOTE** Separate the less sensitive resources, such as dashboards, from the 
more sensitive ones, such as integrations, to avoid having to change tokens.

## Example

The following example shows how to configure the SignalFx provider for Terraform:

```hcl
# Configure the SignalFx provider
provider "signalfx" {
  auth_token = "${var.signalfx_auth_token}"
  # If your organization uses a different realm
  # api_url = "https://api.us2.signalfx.com"
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
* `api_url` - (Optional) The API URL to use for communicating with SignalFx. This is helpful for organizations that need to set their Realm or use a proxy. You can also set it using the `SFX_API_URL` environment variable.
* `custom_app_url` - (Optional) The application URL that users might use to interact with assets in the browser. Used by organizations on specific realms or with a custom [SSO domain](https://docs.signalfx.com/en/latest/admin-guide/sso.html). You can also set it using the `SFX_CUSTOM_APP_URL` environment variable.
* `timeout_seconds` - (Optional) The total timeout duration to wait when making HTTP API calls to SignalFx, in seconds. Defaults to `120`.
