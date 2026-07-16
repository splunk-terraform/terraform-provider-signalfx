resource "signalfx_detector" "test" {
  name              = "demo detector"
  description       = "created by terraform"
  program_text      = "A = data('demo.metric'); detect(when(A > 1)).publish('CPU')"
  timezone          = "Australia/Adelaide"
  max_delay         = 30
  min_delay         = 15
  time_range        = 3600
  show_data_markers = true
  show_event_lines  = true
  disable_sampling  = false

  tags                    = ["resource-tag"]
  teams                   = ["resource-team"]
  authorized_writer_teams = ["writer-team"]
  authorized_writer_users = ["writer-user"]

  rule {
    severity                      = "Critical"
    detect_label                  = "CPU"
    description                   = "CPU is high"
    notifications                 = ["Email,alerts@example.com"]
    disabled                      = false
    parameterized_body            = "body"
    parameterized_subject         = "subject"
    runbook_url                   = "https://example.com/runbook"
    tip                           = "check the service"
    skip_clear_notification_states = ["AUTO_RESOLVED"]

    reminder_notification {
      interval_ms = 60000
      timeout_ms  = 300000
      type        = "TIMEOUT"
    }
  }

  viz_options {
    label        = "CPU"
    color        = "blue"
    display_name = "CPU usage"
    value_unit   = "Byte"
    value_prefix = "$"
    value_suffix = "/s"
  }
}
