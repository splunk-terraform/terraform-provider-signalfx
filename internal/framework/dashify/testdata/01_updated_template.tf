resource "signalfx_dashify_template" "test" {
  template_contents = jsonencode({
    metadata = {
      rootElement = "Chart"
    }
    spec = {
      "<Chart>" = [
        {
          "<o11y:TimeSeriesChart>" = []
          chart                     = {}
          datasource = {
            program    = "A = data('cpu.utilization').publish('A')"
            resolution = 2000  # Updated resolution
          }
        }
      ]
    }
    title = "Updated Test Template"  # Updated title
  })
}

