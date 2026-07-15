resource "signalfx_gcp_integration" "test" {
  name        = "Primary GCP"
  enabled     = true
  poll_rate   = 600
  named_token = "ingest-token"

  services                   = ["compute"]
  custom_metric_type_domains = ["custom.googleapis.com"]
  include_list               = ["labels"]
  exclude_gce_instances_with_labels = ["development", "test"]

  project_service_keys {
    project_id  = "project-a"
    project_key = "secret-a"
  }
  project_service_keys {
    project_id  = "project-b"
    project_key = "secret-b"
  }
}
