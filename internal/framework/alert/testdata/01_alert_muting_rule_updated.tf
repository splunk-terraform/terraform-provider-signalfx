resource "signalfx_alert_muting_rule" "test" {
  description = "updated maintenance"
  detectors   = ["detector-1", "detector-2"]
  start_time  = 1
  stop_time   = 3

  filter {
    property       = "host"
    property_value = "web-2"
    negated        = true
  }

  recurrence {
    unit  = "w"
    value = 1
  }
}
