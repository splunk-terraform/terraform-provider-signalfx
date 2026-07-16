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

  workload_identity_federation_config = "{\"type\":\"external_account\"}"

  projects {
    sync_mode            = "SELECTED"
    selected_project_ids = ["project-a"]
  }
}
