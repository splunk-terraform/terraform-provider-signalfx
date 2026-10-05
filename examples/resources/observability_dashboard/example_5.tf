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

resource "signalfx_observability_dashboard" "advanced_layout" {
  title = "Advanced service layout"

  layout {
    gap  = 8
    step = 8

    defaults {
      min_width  = "4"
      min_height = "2"
    }
  }

  container {
    section {
      title       = "Service health"
      collapse    = false
      collapsible = true

      layout {
        gap  = 4
        step = 4
      }

      container {
        group {
          title      = "Latency details"
          headerless = true

          layout {
            gap  = 0
            step = 1

            defaults {
              height = "20"
            }
          }

          container {
            layout {
              absolute  = true
              width     = jsonencode({ value = "1/2", min = 4, max = "100%" })
              height    = "20"
              min_width = "4"
              max_width = "100%"
              x         = jsonencode(["1/4", 8])
              y         = "12"
            }

            template {
              template_id = signalfx_observability_template.latency.id
            }
          }
        }
      }
    }
  }
}
