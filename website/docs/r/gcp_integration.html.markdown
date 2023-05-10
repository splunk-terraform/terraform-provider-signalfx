---
layout: "signalfx"
page_title: "SignalFx: signalfx_gcp_integration"
sidebar_current: "docs-signalfx-resource-gcp-integration"
description: |-
  Allows Terraform to create and manage SignalFx GCP Integrations
---

# Resource: signalfx_gcp_integration

SignalFx GCP Integration

~> **NOTE** When managing integrations use a session token for an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example Usage

```tf
resource "signalfx_gcp_integration" "gcp_myteam" {
  name      = "GCP - My Team"
  enabled   = true
  poll_rate = 300
  services  = ["compute"]
  project_service_keys {
    project_id  = "gcp_project_id_1"
    project_key = "${file("/path/to/gcp_credentials_1.json")}"
  }
  project_service_keys {
    project_id  = "gcp_project_id_2"
    project_key = "${file("/path/to/gcp_credentials_2.json")}"
  }
}
```

## Argument Reference

* `enabled` - (Required) Whether the integration is enabled.
* `include_list` - (Optional) [Compute Metadata Include List](https://dev.splunk.com/observability/docs/integrations/gcp_integration_overview/).
* `name` - (Required) Name of the integration.
* `named_token` - (Optional) Name of the org token to be used for data ingestion. If not specified then default access token is used.
* `poll_rate` - (Optional) GCP integration poll rate (in seconds). Value between `60` and `600`. Default: `300`.
* `project_service_keys` - (Required) GCP projects to add.
* `services` - (Optional) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'. See the documentation for [Creating Integrations](https://dev.splunk.com/observability/reference/api/integrations/latest#endpoint-create-integration) for valid values.
* `use_metric_source_project_for_quota` - (Optional) When this value is set to true Observability Cloud will force usage of a quota from the project where metrics are stored. For this to work the service account provided for the project needs to be provided with serviceusage.services.use permission or Service Usage Consumer role in this project. When set to false default quota settings are used.
* `whitelist` - (Optional, Deprecated) [Compute Metadata Include List](https://dev.splunk.com/observability/docs/integrations/gcp_integration_overview/).

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
