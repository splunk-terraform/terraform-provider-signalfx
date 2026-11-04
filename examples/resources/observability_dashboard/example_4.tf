resource "signalfx_observability_dashboard" "controls" {
  title = "Service health"

  control_bar = {
    time_range = {
      label   = "Time Range"
      default = "-15m"
    }

    density = {
      default = 60
    }

    pinned_filter = [{
      property         = "service.name"
      label            = "Service"
      default          = ["checkout"]
      suggested_values = ["checkout", "payments"]
      required         = true
      application_mode = "add"
    }]

    filter_set = {
      filter = [{
        property = "deployment.environment"
        values   = ["prod"]
      }]
    }
  }

  container = [{
    layout = {
      width      = "1/2"
      height     = "30"
      min_height = "22"
    }
    metrics_time_series = {
      title   = "Request rate"
      program = "A = data('requests.count').sum().publish('A')"
    }
  }]
}
