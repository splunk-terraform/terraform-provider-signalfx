terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "service_latency" {
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

resource "signalfx_observability_template" "throughput" {
  title        = "Throughput"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('requests.count').sum().publish('A')"
      }
      widget = {
        title = "Throughput"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "service_detail" {
  title = "Service detail"

  container {
    section {
      title = "By service"

      container {
        group {
          title = "Service metrics"

          container {
            layout {
              width  = "6/12"
              height = "2"
            }
            template {
              template_id = signalfx_observability_template.service_latency.id
            }
          }

          container {
            layout {
              width  = "6/12"
              height = "2"
              x      = "6/12"
            }
            template {
              template_id = signalfx_observability_template.throughput.id
            }
          }
        }
      }
    }
  }
}
