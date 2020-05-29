---
layout: "signalfx"
page_title: "SignalFx: signalfx_gcp_integration"
sidebar_current: "docs-signalfx-resource-gcp-integration"
description: |-
  Allows Terraform to create and manage SignalFx GCP Integrations
---

# Resource: signalfx_gcp_integration

SignalFx GCP Integration

~> **NOTE** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider. Otherwise you'll receive a 4xx error.

## Example Usage

```terraform
resource "signalfx_gcp_integration" "gcp_myteam" {
    name = "GCP - My Team"
    enabled = true
    poll_rate = 300000
    services = ["compute"]
    project_service_keys {
        project_id = "gcp_project_id_1"
        project_key = "${file("/path/to/gcp_credentials_1.json")}"
    }
    project_service_keys {
        project_id = "gcp_project_id_2"
        project_key = "${file("/path/to/gcp_credentials_2.json")}"
    }
}
```

## Argument Reference

* `name` - (Required) Name of the integration.
* `enabled` - (Required) Whether the integration is enabled.
* `poll_rate` - (Required) GCP integration poll rate in seconds. Can be set to either 60 or 300 (1 minute or 5 minutes).
* `services` - (Optional) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'. See the documentation for [Creating Integrations](https://developers.signalfx.com/integrations_reference.html#operation/Create%20Integration) for valid values.
* `project_service_keys` - (Required) GCP projects to add.
* `whilelist` - (Optional) [Compute Metadata Whitelist](https://docs.signalfx.com/en/latest/integrations/google-cloud-platform.html#compute-engine-instance).
