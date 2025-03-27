resource "signalfx_org_token" "myteamkey0" {
  name          = "TeamIDKey"
  description   = "My team's rad key"
  notifications = ["Email,foo-alerts@bar.com"]

  host_or_usage_limits {
    host_limit                              = 100
    host_notification_threshold             = 90
    container_limit                         = 200
    container_notification_threshold        = 180
    custom_metrics_limit                    = 1000
    custom_metrics_notification_threshold   = 900
    high_res_metrics_limit                  = 1000
    high_res_metrics_notification_threshold = 900
  }
}
