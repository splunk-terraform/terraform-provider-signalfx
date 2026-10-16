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

resource "signalfx_observability_dashboard" "service" {
  title = "Service overview"

  container = [{
    layout = {
      width  = "50%"
      height = "30"
    }
    template_id = signalfx_observability_template.chart.id
  }]
}
