resource "signalfx_alert_muting_rule" "test" {
  description = "planned maintenance"
  detectors   = ["detector-1", "detector-2"]
  start_time  = 1
  stop_time   = 2

  filter {
    property       = "host"
    property_value = "web-1"
  }

  recurrence {
    unit  = "d"
    value = 2
  }
}
