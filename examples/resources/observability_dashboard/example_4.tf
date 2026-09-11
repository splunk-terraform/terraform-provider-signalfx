terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_template" "cpu_usage" {
  title        = "CPU usage"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('cpu.utilization').mean().publish('A')"
      }
      widget = {
        title = "CPU usage"
      }
    }]
  })
}

resource "signalfx_observability_template" "memory_usage" {
  title        = "Memory usage"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = [{
      "<o11y:SingleValue>" = []
      chart                = {}
      datasource = {
        program = "A = data('memory.utilization').mean().publish('A')"
      }
      widget = {
        title = "Memory usage"
      }
    }]
  })
}

resource "signalfx_observability_dashboard" "infrastructure" {
  title = "Infrastructure overview"

  container {
    group {
      title = "Host resources"

      container {
        layout {
          width = "6/12"
        }
        template {
          template_id = signalfx_observability_template.cpu_usage.id
        }
      }

      container {
        layout {
          width = "6/12"
          x     = "6/12"
        }
        template {
          template_id = signalfx_observability_template.memory_usage.id
        }
      }
    }
  }
}
