resource "signalfx_automated_archival_settings" "setting" {
    enabled         = true
    lookback_period = "P30D"
    grace_period    = "P15D"
}