resource "signalfx_gcp_integration" "test" {
  name        = "Updated GCP"
  enabled     = false
  poll_rate   = 60
  named_token = "ingest-token"
  auth_method = "WORKLOAD_IDENTITY_FEDERATION"

  services                            = ["compute", "storage"]
  custom_metric_type_domains          = ["networking.googleapis.com"]
  include_list                        = ["labels", "metadata"]
  exclude_gce_instances_with_labels   = ["updated"]
  use_metric_source_project_for_quota = true
  import_gcp_metrics                  = false

  project_wif_configs {
    project_id = "project-a"
    wif_config = "{\"audience\":\"legacy\"}"
  }
}
