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
