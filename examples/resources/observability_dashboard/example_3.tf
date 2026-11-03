resource "signalfx_observability_dashboard" "service_layout" {
  title = "Service layout"

  container = [{
    metrics_single_value = {
      title   = "Service latency"
      program = "A = data('latency.p99').mean().publish('A')"
    }
    layout = {
      absolute   = true
      x          = "1/2"
      y          = "2"
      width      = "1/2"
      height     = "15"
      min_width  = "4"
      min_height = "11"
    }
  }]
}
