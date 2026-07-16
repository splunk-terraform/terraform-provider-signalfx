resource "signalfx_gcp_integration" "test" {
  name        = "Invalid GCP"
  enabled     = true
  poll_rate   = 30
  auth_method = "PASSWORD"
  workload_identity_federation_config = "{}"

  project_service_keys {
    project_id  = "project"
    project_key = "secret"
  }

  projects {
    sync_mode = "EVERYTHING"
  }
  projects {
    sync_mode = "SELECTED"
  }
}
