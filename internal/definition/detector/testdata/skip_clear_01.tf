resource "signalfx_detector" "skip_clear" {
  name     = "skip clear notifications detector"
  timezone = "UTC"

  program_text = <<-EOF
    signal = data('app.latency').max().publish('app latency')
    detect(when(signal > 100)).publish('High latency')
  EOF

  rule {
    description                    = "latency above threshold"
    severity                       = "Warning"
    detect_label                   = "High latency"
    notifications                  = ["Email,foo-alerts@example.com"]
    skip_clear_notification_states = ["OK"]
  }
}
