resource "signalfx_observability_template" "chart" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })
}

resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"

  container {
    layout {
      width  = "6/12"
      height = "2"
    }
    template {
      template_id = signalfx_observability_template.chart.id
    }
  }
}
