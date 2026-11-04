resource "signalfx_observability_chart" "service_latency" {
  title = "Reusable service latency"

  metrics_single_value = {
    title   = "Service latency"
    program = "A = data('latency.p99').mean().publish('A')"
  }
}

resource "signalfx_observability_dashboard" "service" {
  title = "Service overview"

  container = [
    {
      metrics_time_series = {
        title         = "Request rate"
        program       = "A = data('requests.count').sum().publish('A')"
        visualization = "line"
      }
      layout = {
        width  = "50%"
        height = "30"
      }
    },
    {
      text = {
        title    = "Runbook"
        markdown = "Check request volume before investigating latency."
      }
      layout = {
        width  = "50%"
        height = "30"
      }
    },
    {
      template_id = signalfx_observability_chart.service_latency.id
      layout = {
        width  = "50%"
        height = "30"
      }
    },
  ]
}
