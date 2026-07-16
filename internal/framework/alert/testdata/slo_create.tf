resource "signalfx_slo" "test" {
  name        = "checkout availability"
  description = "Checkout request success"
  type        = "RequestBased"

  input {
    program_text       = "G = data('good').publish(label='G')\nT = data('total').publish(label='T')"
    good_events_label  = "G"
    total_events_label = "T"
  }

  target {
    type              = "RollingWindow"
    slo               = 98
    compliance_period = "30d"

    alert_rule {
      type = "BREACH"

      rule {
        severity      = "Critical"
        notifications = ["Email,alerts@example.com"]

        parameters {
          fire_lasting = "15m"
        }

        reminder_notification {
          interval_ms = 60000
          timeout_ms  = 120000
          type        = "TIMEOUT"
        }
      }
    }

    alert_rule {
      type = "BURN_RATE"

      rule {
        severity = "Warning"

        parameters {
          short_window_1        = "5m"
          long_window_1         = "1h"
          short_window_2        = "30m"
          long_window_2         = "6h"
          burn_rate_threshold_1 = 14.4
          burn_rate_threshold_2 = 6
        }
      }
    }
  }
}
