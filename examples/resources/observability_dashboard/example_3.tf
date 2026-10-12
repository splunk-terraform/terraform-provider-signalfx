terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "latency" {
  title        = "Service latency"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('latency.p99').mean().publish('A')"
      }
      widget = {
        title = "Service latency"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "service_layout" {
  title = "Service layout"

  container = [{
    template_id = signalfx_observability_template.latency.id
    layout = {
      width      = "1/2"
      height     = "228"
      min_width  = "4"
      min_height = "168"
    }
  }]
}
