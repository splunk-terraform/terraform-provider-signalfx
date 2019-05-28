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

## Example Usage

```hcl
# Configure the SignalFx provider
provider "signalfx" {
  auth_token = "${var.signalfx_auth_token}"
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
* `api_url` - (Optional) The API URL to use for communicating with SignalFx. This is helpful for organizations who need to set their Realm or use a proxy. Note: You likely want to change `custom_app_url` too!
* `custom_app_url` - (Optional)  The application URL that users should use to interact with assets in the browser. This is used by organizations using specific realms or those with a custom [SSO domain](https://docs.signalfx.com/en/latest/admin-guide/sso.html).
