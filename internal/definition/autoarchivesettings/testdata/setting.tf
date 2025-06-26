resource "signalfx_automated_archival_settings" "setting" {
    creator         = "Test Creator"
    last_updated_by = "Test Creator"
    created         = 12340
    last_updated    = 12345
    version         = 1
    enabled         = true
    lookback_period = "P30D"
    grace_period    = "P15D"
    ruleset_limit   = 1000
}