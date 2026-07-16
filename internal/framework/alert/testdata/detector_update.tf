resource "signalfx_detector" "test" {
  name               = "updated detector"
  description        = "updated by terraform"
  program_text       = "A = data('demo.metric'); detect(when(A > 2)).publish('CPU')"
  timezone           = "UTC"
  max_delay          = 60
  min_delay          = 30
  start_time         = 1700000000
  end_time           = 1700003600
  show_data_markers  = false
  show_event_lines   = false
  disable_sampling   = true

  tags                    = ["updated-tag"]
  teams                   = ["updated-team"]
  authorized_writer_teams = ["updated-writer-team"]
  authorized_writer_users = ["updated-writer-user"]

  rule {
    severity      = "Warning"
    detect_label  = "CPU"
    description   = "CPU remains high"
    notifications = ["Email,updated@example.com"]
    disabled      = true
    parameterized_body    = ""
    parameterized_subject = ""
    runbook_url           = ""
    tip                   = ""
  }

  viz_options {
    label        = "CPU"
    color        = "red"
    display_name = "Updated CPU"
    value_unit   = "Second"
    value_prefix = ""
    value_suffix = ""
  }
}
