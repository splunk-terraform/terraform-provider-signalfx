---
layout: "signalfx"
page_title: "SignalFx: signalfx_resource"
sidebar_current: "docs-signalfx-resource-gcp-integration"
description: |-
  Allows Terraform to create and manage SignalFx GCP Integrations
---

# Resource: signalfx_integration_gcp

SignalFx GCP Integration

## Example Usage

```terraform
resource "signalfx_integration_gcp" "gcp_myteam" {
    name = "GCP - My Team"
    enabled = true
    poll_rate = 300000
    services = ["compute"]
    project_service_keys
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
* `poll_rate` - (Required) GCP integration poll rate in milliseconds. Can be set to either 60000 or 300000 (1 minute or 5 minutes).
* `services` - (Optional) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'.
* `project_service_keys` - (Required) GCP projects to add.
