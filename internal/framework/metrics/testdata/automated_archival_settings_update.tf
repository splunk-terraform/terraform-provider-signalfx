resource "signalfx_automated_archival_settings" "test" {
  enabled         = false
  lookback_period = "P45D"
  grace_period    = "P30D"
  ruleset_limit   = 20
}
