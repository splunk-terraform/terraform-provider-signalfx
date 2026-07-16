resource "signalfx_automated_archival_exempt_metric" "test" {
  exempt_metrics {
    name = "metric.cpu"
  }

  exempt_metrics {
    name = "metric.memory"
  }
}
