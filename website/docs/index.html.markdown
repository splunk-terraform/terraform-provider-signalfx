---
layout: "signalfx"
page_title: "Provider: SignalFx"
sidebar_current: "docs-signalfx-index"
description: |-
  The SignalFx provider is used to interact with the resources supported by SignalFx. The provider needs to be configured with the proper credentials before it can be used.
---

# SignalFx Provider

The [SignalFx](https://www.signalfx.com/) provider is used to interact with the
resources supported by SignalFx. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

# Learning

If you're new to Terraform and SignalFx there are few resource we can offer to
help!

Terraform is a tool and ecosystem beyond just SignalFx. It's really powerful and you can check out an [Introduction to Terraform](https://www.terraform.io/intro/index.html)
which covers the basic usage.

Once you got the basics of working with Terraform down, using this provider is much easier. You'll probably want to check out the docs on the [SignalFlow programming language](https://developers.signalfx.com/signalflow_analytics/signalflow_overview.html#_signalflow_programming_language) as all charts and detectors will require you to provide the `program_text` in SignalFlow. Also keep in mind that you can open any chart or in SignalFx and click "Show SignalFlow" if you prefer to use the UI.

## Example Usage

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

## Argument Reference

The following arguments are supported:

* `auth_token` - (Required) The auth token for [authentication](https://developers.signalfx.com/basics/authentication.html). This can also be set via the `SFX_AUTH_TOKEN` environment variable.
* `api_url` - (Optional) The API URL to use for communicating with SignalFx. This is helpful for organizations who need to set their Realm or use a proxy. Note: You likely want to change `custom_app_url` too! This can also be set via the `SFX_API_URL` environment variable.
* `custom_app_url` - (Optional)  The application URL that users should use to interact with assets in the browser. This is used by organizations using specific realms or those with a custom [SSO domain](https://docs.signalfx.com/en/latest/admin-guide/sso.html). This can also be set via the `SFX_CUSTOM_APP_URL` environment variable.
* `timeout_seconds` - (Optional) The total timeout duration, in seconds, to wait when making HTTP API calls to SignalFx. Defaults to 120.
