terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "control_chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('requests.count').sum().publish('A')"
      }
      widget = {
        title = "Request rate"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "controls" {
  title = "Service health"

  control_bar {
    time_range {
      label                  = "Time Range"
      default_variable_value = "-15m"
    }

    density {
      default_variable_value = 60
    }

    pinned_filter {
      variable_name          = "service"
      key                    = "service.name"
      default_variable_value = ["checkout"]
      suggested_values       = ["checkout", "payments"]
      required               = true
      application_mode       = "add"
    }

    filter_set {
      hidden = false

      filter {
        key    = "deployment.environment"
        values = ["prod"]
      }
    }
  }

  container {
    template {
      template_id = signalfx_observability_template.control_chart.id
    }
  }
}
