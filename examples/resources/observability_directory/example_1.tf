terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

variable "user_email" {
  type        = string
  description = "Email address of the user whose Directory namespace is managed."
}

resource "signalfx_observability_template" "chart" {
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

resource "signalfx_observability_directory" "charts" {
  path   = "~users/${var.user_email}/charts"
  pinned = true
  templates = [
    signalfx_observability_template.chart.id,
  ]
}
