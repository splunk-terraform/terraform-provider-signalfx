provider "signalfx" {}

resource "signalfx_team" "my_detector_team" {
  name        = "example team"
  description = "A team made from terraform"

  notifications_default = ["Email,test@example.com"]
}

resource "signalfx_detector" "my_detector" {
  name            = "example detector"
  description     = "A detector made from terraform"
  max_delay       = 30
  min_delay       = 15
  tags            = ["tag-1", "tag-2"]
  teams           = [signalfx_team.my_detector_team.id]
  timezone        = "Europe/Paris"
  detector_origin = "Standard"

  program_text = <<-EOF
    signal = data('app.delay').max().publish('app delay')
    detect(when(signal > 60, '5m')).publish('Processing old messages 5m')
    detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
  EOF

  rule {
    description   = "maximum > 60 for 5m"
    severity      = "Warning"
    detect_label  = "Processing old messages 5m"
    notifications = ["Email,foo-alerts@example.com"]
  }

  rule {
    description   = "maximum > 60 for 30m"
    severity      = "Critical"
    detect_label  = "Processing old messages 30m"
    notifications = ["Email,foo-alerts@example.com"]
  }

  viz_options {
    label      = "app delay"
    color      = "orange"
    value_unit = "Second"
  }
}
