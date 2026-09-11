terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "request_rate" {
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

resource "signalfx_observability_template" "error_rate" {
  title        = "Error rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('errors.count').sum().publish('A')"
      }
      widget = {
        title = "Error rate"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "service_health" {
  title = "Service health"

  container {
    section {
      title = "Traffic"

      container {
        layout {
          width  = "6/12"
          height = "2"
          x      = "0"
        }
        template {
          template_id = signalfx_observability_template.request_rate.id
        }
      }

      container {
        layout {
          width  = "6/12"
          height = "2"
          x      = "6/12"
        }
        template {
          template_id = signalfx_observability_template.error_rate.id
        }
      }
    }
  }
}
