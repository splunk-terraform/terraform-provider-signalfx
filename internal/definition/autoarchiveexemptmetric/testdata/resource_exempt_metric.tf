resource "signalfx_automated_archival_exempt_metric" "exempt_metric" {
    exempt_metrics {
        name = "exempt_metric_1"
    }
}