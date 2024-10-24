resource "signalfx_detector" "my_detector" {
  name        = "max average delay UPDATED"
  description = "your application is slowER"
  max_delay   = 60
  min_delay   = 30
  timezone    = "Europe/Paris"

  show_data_markers = true
  show_event_lines  = true
  disable_sampling  = true
  time_range        = 3600
  tags              = ["tag-1", "tag-2", "tag-3"]

  program_text = <<-EOF
    signal = data('app.delay2').max().publish('app delay')
    detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
    detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
  EOF

  rule {
    description   = "NEW maximum > 60 for 5m"
    severity      = "Warning"
    detect_label  = "Processing old messages 5m"
    notifications = ["Email,foo-alerts@example.com"]
    runbook_url   = "https://www.example.com"
    tip           = "reboot it"
  }

  rule {
    description   = "NEW maximum > 60 for 30m"
    severity      = "Critical"
    detect_label  = "Processing old messages 30m"
    notifications = ["Email,foo-alerts@example.com"]
    runbook_url   = "https://www.example.com"
  }

  viz_options {
    label      = "app delay"
    color      = "orange"
    value_unit = "Second"
  }
}
