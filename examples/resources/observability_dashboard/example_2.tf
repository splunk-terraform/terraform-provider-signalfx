resource "signalfx_observability_dashboard" "inline_content" {
  title = "Inline dashboard content"

  container = [{
    layout = {
      width      = "100%"
      height     = "30"
      min_height = "22"
    }

    inline_content = jsonencode({
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
  }]
}
