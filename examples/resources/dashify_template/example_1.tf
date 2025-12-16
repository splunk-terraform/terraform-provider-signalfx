# Create a Dashify template
resource "signalfx_dashify_template" "example_template" {
  template_contents = jsonencode({
    metadata = {
      imports = []
      datasource = {
        type        = "SPLUNK_O11Y"
        programText = "A = data('metric').publish(label='A')"
      }
      rootElement = "dashboard"
    }
    spec = {
      "<Dashboard>" = [
        {
          "<o11y:ApmServiceMap>" = []
          chart                   = {}
          datasource = {
            operationName = "Graph"
            variables = {
              filters = {
                tags = [
                  {
                    tagName = "sf_service"
                    "values$" = "$state.FILTERS().sf_service?.values"
                  }
                ]
              }
              timeRange = {
                "endTimestampMillis$" = "$state.TIME().now"
                lookbackMillis        = 691200000
              }
            }
          }
        }
      ]
    }
  })
}

# Output the template ID
output "template_id" {
  value = signalfx_dashify_template.example_template.id
}

