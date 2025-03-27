resource "signalfx_dashboard" "mydashboard0" {
  name            = "My Dashboard"
  dashboard_group = signalfx_dashboard_group.mydashboardgroup0.id

  time_range = "-30m"

  filter {
    property = "collector"
    values   = ["cpu", "Diamond"]
  }
  variable {
    property = "region"
    alias    = "region"
    values   = ["uswest-1-"]
  }
  chart {
    chart_id = signalfx_time_chart.mychart0.id
    width    = 12
    height   = 1
  }
  chart {
    chart_id = signalfx_time_chart.mychart1.id
    width    = 5
    height   = 2
  }
}
