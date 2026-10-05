resource "signalfx_observability_dashboard" "inline_content" {
  title = "Inline dashboard content"

  container {
    layout {
      width  = "6/12"
      height = "2"
    }

    template {
      content = jsonencode({
        "<o11y:SingleValue>" = []
        chart = {
          color = "blue"
        }
        datasource = {
          program = "A = data('requests.count').sum().publish('A')"
        }
        widget = {
          title = "Request rate"
        }
      })
    }
  }
}
