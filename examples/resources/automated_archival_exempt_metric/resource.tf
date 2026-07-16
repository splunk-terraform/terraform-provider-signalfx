resource "signalfx_automated_archival_exempt_metric" "example" {
  exempt_metrics {
    name = "service.request.count"
  }

  exempt_metrics {
    name = "service.request.duration"
  }
}
