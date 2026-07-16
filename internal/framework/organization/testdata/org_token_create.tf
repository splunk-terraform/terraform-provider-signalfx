resource "signalfx_org_token" "test" {
  name        = "primary-token"
  description = "Primary token"
  auth_scopes = ["API", "INGEST"]

  host_or_usage_limits {
    host_limit                     = 100
    host_notification_threshold    = 90
    custom_metrics_limit           = 2000
    custom_metrics_notification_threshold = 1800
  }

  notifications = ["Email,alerts@example.com"]
}
