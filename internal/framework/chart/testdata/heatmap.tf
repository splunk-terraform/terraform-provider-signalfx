resource "signalfx_chart" "heatmap" {
  name        = "heatmap chart"
  description = "An example of a heatmap chart"
  tags = [
    "tag-00",
    "tag-01"
  ]
  
  heatmap = {
    color_by = "Range"
    group_by = ["dimension-00"]
  }


  program_options = {
    text             = <<-EOT
    A = data('value').sum(by='dimension-00').publish('A')
    EOT
    min_resolution   = "1m"
    max_delay        = "5m"
    disable_sampling = false
    timezone         = "UTC"
  }

  data_options = {
    no_data = {
      message = "You need to enable value receiver as part of your otel distribution"
    }
    refresh_interval = "1m"
    max_precision    = 3
    time_range       = "40h"
    unit_prefix      = "Metric"
  }
}
