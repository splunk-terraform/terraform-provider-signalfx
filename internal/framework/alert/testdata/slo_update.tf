resource "signalfx_slo" "test" {
  name        = "checkout availability"
  description = "Updated checkout request success"
  type        = "RequestBased"

  input {
    program_text       = "G = data('good-v2').publish(label='G')\nT = data('total').publish(label='T')"
    good_events_label  = "G"
    total_events_label = "T"
  }

  target {
    type              = "RollingWindow"
    slo               = 99
    compliance_period = "30d"

    alert_rule {
      type = "BREACH"

      rule {
        severity      = "Critical"
        notifications = ["Email,new-alerts@example.com"]

        parameters {
          fire_lasting = "5m"
        }
      }
    }

    alert_rule {
      type = "ERROR_BUDGET_LEFT"

      rule {
        severity = "Warning"

        parameters {
          percent_error_budget_left = 12
        }
      }
    }
  }
}
