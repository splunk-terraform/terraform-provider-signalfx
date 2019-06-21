---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-integration"
description: |-
  Allows Terraform to create and manage SignalFx Integrations
---

# Resource: signalfx_integration

SignalFx supports integrations to ingest metrics from other monitoring systems, connect to Single Sign-On providers, and to report notifications for messaging and incident management. Note that your API key must have admin permissions to use the SignalFx integration API.

**Note:** This resource is deprecated.

In a future release `signalfx_integration` will be replaced with specific resources for each integration. Please see the specific `signalfx_integration_*` resources in the sidebar.


## Example Usage

### PagerDuty
```terraform
resource "signalfx_integration" "pagerduty_myteam" {
    name = "PD - My Team"
    enabled = true
    type = "PagerDuty"
    api_key = "1234567890"
}
```

### Slack
```terraform
resource "signalfx_integration" "slack_myteam" {
    name = "Slack - My Team"
    enabled = true
    type = "Slack"
    webhook_url = "http://example.com"
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `type` - (Required) Type of the integration. See [the full list here](https://developers.signalfx.com/integrations_reference.html).
* `api_key` - (Required for `PagerDuty`) PagerDuty API key.
* `webhook_url` - (Required for `Slack`) Slack incoming webhook URL.
* `poll_rate` - (Required for `GCP`) GCP integration poll rate in milliseconds. Can be set to either 60000 or 300000 (1 minute or 5 minutes).
* `services` - (Optional for `GCP`) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'.
* `project_service_keys` - (Required for `GCP`) GCP projects to add.
