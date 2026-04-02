resource "signalfx_alert_muting_rule" "rool_mooter_one" {
  description = "mooted it NEW"

  start_time = 1573063243
  stop_time  = 0 # Defaults to 0

  detectors = [signalfx_detector.some_detector.id]

  filter {
    property       = "foo"
    property_value = "bar"
  }

  recurrence {
    unit = "d"
    value = 2
  }
}
