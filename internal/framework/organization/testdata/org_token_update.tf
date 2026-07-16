resource "signalfx_org_token" "test" {
  name        = "updated-token"
  description = "Updated token"
  auth_scopes = ["INGEST"]
  disabled    = true

  dpm_limits {
    dpm_limit                  = 5000
    dpm_notification_threshold = 4500
  }

  notifications = ["Slack,credential-id,token-alerts"]
}
