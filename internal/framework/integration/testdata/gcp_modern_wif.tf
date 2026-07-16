resource "signalfx_gcp_integration" "test" {
  name        = "Modern WIF GCP"
  enabled     = true
  poll_rate   = 300
  named_token = "ingest-token"
  auth_method = "WORKLOAD_IDENTITY_FEDERATION"

  services                              = []
  workload_identity_federation_config   = "{\"type\":\"external_account\"}"
  use_metric_source_project_for_quota   = false
  import_gcp_metrics                    = true

  projects {
    sync_mode           = "SELECTED"
    selected_project_ids = ["project-b", "project-a"]
  }
}
