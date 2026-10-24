resource "signalfx_observability_chart" "request_rate" {
  title = "Reusable request rate"

  metrics_time_series = {
    title         = "Request rate"
    program       = "A = data('requests.count').sum().publish('A')"
    time_range    = "1h"
    visualization = "line"

    series = {
      A = {
        name = "Requests"
        display_unit = {
          suffix = "/s"
        }
      }
    }
  }
}

resource "signalfx_observability_dashboard" "service" {
  title = "Service overview"

  container = [{
    template_id = signalfx_observability_chart.request_rate.id
  }]
}
