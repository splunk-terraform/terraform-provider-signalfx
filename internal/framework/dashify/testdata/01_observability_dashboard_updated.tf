resource "signalfx_observability_template" "chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })
}

resource "signalfx_observability_dashboard" "test" {
  title = "Updated service overview"

  container {
    template {
      template_id = signalfx_observability_template.chart.id
    }
  }
}
