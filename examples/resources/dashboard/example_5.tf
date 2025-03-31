resource "signalfx_dashboard" "load" {
  name            = "Load"
  dashboard_group = signalfx_dashboard_group.example.id

  column {
    chart_ids = [signalfx_single_value_chart.rps.*.id]
    width     = 2
  }
  column {
    chart_ids = [signalfx_time_chart.cpu_capacity.*.id]
    column    = 2
    width     = 4
  }
}
