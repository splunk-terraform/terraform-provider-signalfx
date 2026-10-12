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

resource "signalfx_observability_dashboard" "nested_layout" {
  title = "Service layout"

  layout = {
    gap  = 8
    step = 8

    defaults = {
      min_width  = "4"
      height     = "40"
      min_height = "32"
    }
  }

  container = [{
    section = {
      title       = "Service health"
      collapse    = false
      collapsible = true

      layout = {
        gap  = 4
        step = 4
      }

      container = [{
        group = {
          title      = "Latency details"
          headerless = true

          layout = {
            gap  = 8
            step = 1

            defaults = {
              height     = "228"
              min_height = "168"
            }
          }

          container = [{
            layout = {
              width      = "1/2"
              height     = "228"
              min_height = "168"
              min_width  = "4"
              max_width  = "100%"
            }

            template_id = signalfx_observability_template.latency.id
          }]
        }
      }]
    }
  }]
}
