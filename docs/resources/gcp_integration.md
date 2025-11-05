---
page_title: "Splunk Observability Cloud: signalfx_gcp_integration"
description: |-
  Allows Terraform to create and manage GCP Integrations for Splunk Observability Cloud
---

# Resource: signalfx_gcp_integration

Splunk Observability Cloud GCP Integration.

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```terraform
resource "signalfx_gcp_integration" "gcp_myteam" {
  name                       = "GCP - My Team"
  enabled                    = true
  poll_rate                  = 300
  services                   = ["compute"]
  custom_metric_type_domains = ["istio.io"]
  import_gcp_metrics         = true
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

## Arguments

* `custom_metric_type_domains` - (Optional) List of additional GCP service domain names that Splunk Observability Cloud will monitor. See [Custom Metric Type Domains documentation](https://dev.splunk.com/observability/docs/integrations/gcp_integration_overview/#Custom-metric-type-domains)
* `enabled` - (Required) Whether the integration is enabled.
* `import_gcp_metrics` - (Optional) If enabled, Splunk Observability Cloud will sync also Google Cloud Monitoring data. If disabled, Splunk Observability Cloud will import only metadata. Defaults to true.
* `include_list` - (Optional) [Compute Metadata Include List](https://dev.splunk.com/observability/docs/integrations/gcp_integration_overview/).
* `name` - (Required) Name of the integration.
* `named_token` - (Optional) Name of the org token to be used for data ingestion. If not specified then default access token is used.
* `poll_rate` - (Optional) GCP integration poll rate (in seconds). Value between `60` and `600`. Default: `300`.
* `project_service_keys` - (Optional) GCP projects to add.
* `services` - (Optional) GCP service metrics to import. Can be an empty list, or not included, to import 'All services'. See [Google Cloud Platform services](https://docs.splunk.com/Observability/gdi/get-data-in/integrations.html#google-cloud-platform-services) for a list of valid values.
* `use_metric_source_project_for_quota` - (Optional) When this value is set to true Observability Cloud will force usage of a quota from the project where metrics are stored. For this to work the service account provided for the project needs to be provided with serviceusage.services.use permission or Service Usage Consumer role in this project. When set to false default quota settings are used.
* `workload_identity_federation_config` - (Optional) Your Workload Identity Federation config. To easily set up WIF you can use helpers provided in the [gcp_workload_identity_federation](https://github.com/signalfx/gcp_workload_identity_federation/tree/main/terraform) repository.
* `projects` - (Optional) Object comprised of `sync_mode` and optional `selected_project_ids`. If you use `sync_mode` `ALL_REACHABLE` then Splunk Observability Cloud will automatically discover GCP projects that the provided WIF principal has permissions to query. If `sync_mode` is `SELECTED`, you need to provide a list of project ids in the `selected_project_ids` field.
* `project_wif_configs` (Deprecated) Please use `workload_identity_federation_config` with `projects` instead.

## Attributes

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration.
