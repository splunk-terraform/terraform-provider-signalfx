resource "signalfx_slo" "foo_service_slo" {
  name        = "foo service SLO"
  type        = "RequestBased"
  description = "SLO monitoring for foo service"

  input {
    program_text       = "G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_service', 'foo-service'))\nT = data('spans.count', filter=filter('sf_service', 'foo-service'))"
    good_events_label  = "G"
    total_events_label = "T"
  }

  target {
    type              = "RollingWindow"
    slo               = 95
    compliance_period = "30d"

    alert_rule {
      type = "BREACH"

      rule {
        severity = "Warning"
        notifications = ["Email,foo-alerts@bar.com"]
      }
    }
  }
}

provider "signalfx" {}

