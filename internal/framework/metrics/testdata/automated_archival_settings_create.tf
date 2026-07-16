resource "signalfx_automated_archival_settings" "test" {
  enabled         = true
  lookback_period = "P30D"
  grace_period    = "P15D"
  ruleset_limit   = 10
}
