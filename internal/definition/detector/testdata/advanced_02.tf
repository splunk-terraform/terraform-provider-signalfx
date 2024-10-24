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

  program_text = <<-EOF
    signal = data('app.delay2').max().publish('app delay')
    detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
  EOF

  rule {
    description   = "NEW maximum > 60 for 5m"
    severity      = "Warning"
    detect_label  = "Processing old messages 5m"
    notifications = ["Email,foo-alerts@example.com"]
    runbook_url   = "https://www.example.com"
    tip           = "reboot it"
  }

  viz_options {
    label      = "app delay"
    color      = "orange"
    value_unit = "Second"
  }

}
